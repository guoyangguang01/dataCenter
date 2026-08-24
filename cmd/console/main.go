package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "github.com/datacenter/api/v1"
	"github.com/datacenter/internal/alert"
	"github.com/datacenter/internal/device"
	"github.com/datacenter/internal/domain"
	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/modbus"
	"github.com/datacenter/internal/mqtt"
	"github.com/datacenter/internal/opcua"
	"github.com/datacenter/internal/model"
	"github.com/datacenter/internal/rule"
	"github.com/datacenter/internal/tcp"
	"github.com/datacenter/internal/timeseries"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 数据库
	db, err := gorm.Open(sqlite.Open("datacenter.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// AutoMigrate 所有表
	if err := db.AutoMigrate(
		&device.Device{},
		&domain.Domain{},
		&domain.DomainMember{},
		&model.ThingModel{},
		&model.DeviceModelBinding{},
		&rule.RuleConfig{},
		&alert.WebhookConfig{},
		&alert.AlertLog{},
		&gateway.GatewayConfig{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// 首次启动时填充种子数据
	seedData(db)

	// 服务层
	deviceService := device.NewService(db)
	domainService := domain.NewService(db)
	modelService := model.NewService(db)
	alertService := alert.NewAlertService(db)
	gatewayService := gateway.NewGatewayService(db)

	// 规则引擎
	registry := rule.NewRegistry()
	rule.RegisterBuiltinNodes(registry, nil)
	rule.RegisterScriptNode(registry)
	ruleEngine := rule.NewEngine(registry)

	ruleConfigService := rule.NewRuleConfigService(db, ruleEngine)
	if err := ruleConfigService.InitDB(); err != nil {
		log.Fatalf("failed to init rule config db: %v", err)
	}
	if err := ruleConfigService.LoadAll(); err != nil {
		log.Printf("warning: failed to load rules: %v", err)
	}

	// 网关启动器（优先使用 NATS，不可用时降级为日志 publisher）
	fmt.Println("[Console] 🔗 尝试连接 NATS...")
	var gwPublisher gateway.Publisher
	nc, natsErr := nats.Connect("nats://localhost:4222", nats.Name("console-gateway"))
	if natsErr != nil {
		fmt.Printf("[Console] ⚠️ NATS 不可用，使用 LogPublisher: %v\n", natsErr)
		gwPublisher = gateway.NewLogPublisher()
	} else {
		gwPublisher = &gateway.SimpleNATSPublisher{Conn: nc}
		fmt.Println("[Console] ✅ NATS 已连接: nats://localhost:4222")
	}
	launcher := gateway.NewLauncher(gatewayService, gwPublisher)

	// 注册网关工厂
	launcher.Register(gateway.TypeMQTT, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseMQTTConfig(configStr)
		if err != nil {
			return nil, err
		}
		gw := mqtt.NewGateway(mqtt.Config{
			Port:          cfg.Port,
			MaxConnection: cfg.MaxConnection,
			KeepAlive:     cfg.KeepAlive,
		}, pub)
		// 设置回调：收到数据时标记设备在线
		gw.SetOnDataReceived(func(deviceID string) {
			fmt.Printf("[Callback] marking device online: %s\n", deviceID)
			result := db.Model(&device.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
				"online":    true,
				"last_seen": time.Now(),
			})
			fmt.Printf("[Callback] rows affected: %d\n", result.RowsAffected)
		})
		return gw, nil
	})

	launcher.Register(gateway.TypeTCP, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseTCPConfig(configStr)
		if err != nil {
			return nil, err
		}
		gw := tcp.NewGateway(tcp.Config{
			Port:          cfg.Port,
			MaxConnection: cfg.MaxConnection,
			Heartbeat:     cfg.Heartbeat,
		}, pub)
		gw.SetOnDataReceived(func(deviceID string) {
			db.Model(&device.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
				"online":    true,
				"last_seen": time.Now(),
			})
		})
		return gw, nil
	})

	launcher.Register(gateway.TypeModbus, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseModbusConfig(configStr)
		if err != nil {
			return nil, err
		}
		gw := modbus.NewGateway(modbus.Config{
			Port:         cfg.Port,
			PollInterval: cfg.PollInterval,
			SlaveIDs:     cfg.SlaveIDs,
		}, pub)
		gw.SetOnDataReceived(func(deviceID string) {
			db.Model(&device.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
				"online":    true,
				"last_seen": time.Now(),
			})
		})
		return gw, nil
	})

	launcher.Register(gateway.TypeOPCUA, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseOPCUAConfig(configStr)
		if err != nil {
			return nil, err
		}
		return opcua.NewGateway(opcua.Config{
			Endpoint:     cfg.Endpoint,
			PollInterval: cfg.PollInterval,
			NodeIDs:      cfg.NodeIDs,
			DeviceID:     cfg.DeviceID,
			DomainID:     cfg.DomainID,
		}, pub), nil
	})

	// Gin
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// 路由
	api := r.Group("/api/v1")
	v1.NewDeviceHandler(deviceService).RegisterRoutes(api)
	v1.NewDomainHandler(domainService).RegisterRoutes(api)
	v1.NewModelHandler(modelService).RegisterRoutes(api)
	v1.NewRuleHandler(ruleConfigService).RegisterRoutes(api)
	v1.NewAlertHandler(alertService).RegisterRoutes(api)
	v1.NewGatewayHandler(gatewayService, launcher).RegisterRoutes(api)
	v1.NewStatsHandler(deviceService, alertService, gatewayService, launcher).RegisterRoutes(api)

	// 时序数据查询（TDengine 不可用时优雅降级）
	tsQueryService, err := timeseries.NewQueryService(timeseries.Config{DSN: "root:taosdata@http(localhost:6041)/"})
	if err != nil {
		log.Printf("warning: TDengine not available: %v (data queries will be disabled)", err)
	}
	v1.NewDataHandler(tsQueryService).RegisterRoutes(api)

	// 启动 HTTP 服务
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		fmt.Println("Console server listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down...")

	// 停止所有网关
	launcher.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	fmt.Println("Server exited")
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func seedData(db *gorm.DB) {
	// 检查是否已有数据，避免重复填充
	var domainCount int64
	db.Model(&domain.Domain{}).Count(&domainCount)
	if domainCount > 0 {
		return
	}

	log.Println("首次启动，填充种子数据...")

	// 1. 业务域
	domains := []domain.Domain{
		{ID: "factory-01", Name: "一号工厂", Description: "华东地区主要生产基地"},
		{ID: "factory-02", Name: "二号工厂", Description: "华南地区分厂"},
	}
	for _, d := range domains {
		db.Create(&d)
	}
	// 域成员
	db.Create(&domain.DomainMember{DomainID: "factory-01", UserID: "admin", Role: domain.RoleSuperAdmin})
	db.Create(&domain.DomainMember{DomainID: "factory-02", UserID: "admin", Role: domain.RoleSuperAdmin})

	// 2. 物模型
	models := []model.ThingModel{
		// factory-01 物模型
		{
			ID: "model-thermometer", Name: "温湿度传感器", DomainID: "factory-01",
			Properties: []model.PropertyDef{
				{ID: "temperature", Name: "温度", DataType: "float", Unit: "°C", Range: [2]float64{-40, 80}, Required: true, AccessMode: "r"},
				{ID: "humidity", Name: "湿度", DataType: "float", Unit: "%RH", Range: [2]float64{0, 100}, Required: true, AccessMode: "r"},
				{ID: "battery", Name: "电池电量", DataType: "float", Unit: "%", Range: [2]float64{0, 100}, Required: false, AccessMode: "r"},
			},
			Commands: []model.CommandDef{
				{ID: "set_interval", Name: "设置上报间隔", Params: []model.ParamDef{{ID: "seconds", Name: "间隔秒数", DataType: "int"}}},
			},
			Events: []model.EventDef{
				{ID: "temp_alert", Name: "温度告警", Params: []model.ParamDef{{ID: "value", Name: "当前温度", DataType: "float"}}},
			},
		},
		{
			ID: "model-motor", Name: "电机监控模型", DomainID: "factory-01",
			Properties: []model.PropertyDef{
				{ID: "rpm", Name: "转速", DataType: "float", Unit: "rpm", Range: [2]float64{0, 3000}, Required: true, AccessMode: "r"},
				{ID: "vibration", Name: "振动", DataType: "float", Unit: "mm/s", Range: [2]float64{0, 50}, Required: true, AccessMode: "r"},
				{ID: "temperature", Name: "电机温度", DataType: "float", Unit: "°C", Range: [2]float64{0, 150}, Required: true, AccessMode: "r"},
			},
			Commands: []model.CommandDef{
				{ID: "start", Name: "启动电机", Params: nil},
				{ID: "stop", Name: "停止电机", Params: nil},
				{ID: "set_rpm", Name: "设定转速", Params: []model.ParamDef{{ID: "target_rpm", Name: "目标转速", DataType: "float"}}},
			},
			Events: []model.EventDef{
				{ID: "overheat", Name: "过热告警", Params: []model.ParamDef{{ID: "temp", Name: "当前温度", DataType: "float"}}},
				{ID: "vibration_alert", Name: "振动异常", Params: []model.ParamDef{{ID: "value", Name: "振动值", DataType: "float"}}},
			},
		},
		{
			ID: "model-power-meter", Name: "智能电表", DomainID: "factory-01",
			Properties: []model.PropertyDef{
				{ID: "voltage", Name: "电压", DataType: "float", Unit: "V", Range: [2]float64{0, 500}, Required: true, AccessMode: "r"},
				{ID: "current", Name: "电流", DataType: "float", Unit: "A", Range: [2]float64{0, 100}, Required: true, AccessMode: "r"},
				{ID: "power", Name: "功率", DataType: "float", Unit: "kW", Range: [2]float64{0, 50}, Required: true, AccessMode: "r"},
				{ID: "energy", Name: "累计电量", DataType: "float", Unit: "kWh", Range: [2]float64{0, 999999}, Required: false, AccessMode: "r"},
			},
		},
		// factory-02 物模型
		{
			ID: "model-gas-sensor", Name: "气体检测传感器", DomainID: "factory-02",
			Properties: []model.PropertyDef{
				{ID: "co2", Name: "CO2浓度", DataType: "float", Unit: "ppm", Range: [2]float64{0, 5000}, Required: true, AccessMode: "r"},
				{ID: "co", Name: "CO浓度", DataType: "float", Unit: "ppm", Range: [2]float64{0, 500}, Required: true, AccessMode: "r"},
				{ID: "temperature", Name: "环境温度", DataType: "float", Unit: "°C", Range: [2]float64{-20, 60}, Required: true, AccessMode: "r"},
			},
			Events: []model.EventDef{
				{ID: "gas_leak", Name: "气体泄漏告警", Params: []model.ParamDef{{ID: "gas_type", Name: "气体类型", DataType: "string"}, {ID: "concentration", Name: "浓度", DataType: "float"}}},
			},
		},
		{
			ID: "model-water-meter", Name: "智能水表", DomainID: "factory-02",
			Properties: []model.PropertyDef{
				{ID: "flow_rate", Name: "瞬时流量", DataType: "float", Unit: "m³/h", Range: [2]float64{0, 100}, Required: true, AccessMode: "r"},
				{ID: "total_flow", Name: "累计流量", DataType: "float", Unit: "m³", Range: [2]float64{0, 999999}, Required: true, AccessMode: "r"},
				{ID: "pressure", Name: "管道压力", DataType: "float", Unit: "MPa", Range: [2]float64{0, 2.5}, Required: false, AccessMode: "r"},
			},
		},
	}
	for _, m := range models {
		db.Create(&m)
	}

	// 3. 设备
	devices := []device.Device{
		// factory-01 设备
		{ID: "th_sensor_001", Name: "车间A温湿度传感器1号", DeviceType: "sensor", Protocol: "mqtt", DomainID: "factory-01", Region: "workshop-a", ModelID: "model-thermometer", Token: generateToken()},
		{ID: "th_sensor_002", Name: "车间A温湿度传感器2号", DeviceType: "sensor", Protocol: "mqtt", DomainID: "factory-01", Region: "workshop-a", ModelID: "model-thermometer", Token: generateToken()},
		{ID: "th_sensor_003", Name: "车间B温湿度传感器1号", DeviceType: "sensor", Protocol: "mqtt", DomainID: "factory-01", Region: "workshop-b", ModelID: "model-thermometer", Token: generateToken()},
		{ID: "motor_001", Name: "车间A电机1号", DeviceType: "actuator", Protocol: "mqtt", DomainID: "factory-01", Region: "workshop-a", ModelID: "model-motor", Token: generateToken()},
		{ID: "motor_002", Name: "车间A电机2号", DeviceType: "actuator", Protocol: "mqtt", DomainID: "factory-01", Region: "workshop-a", ModelID: "model-motor", Token: generateToken()},
		{ID: "motor_003", Name: "车间B电机1号", DeviceType: "actuator", Protocol: "tcp", DomainID: "factory-01", Region: "workshop-b", ModelID: "model-motor", Token: generateToken()},
		{ID: "meter_001", Name: "配电室智能电表", DeviceType: "sensor", Protocol: "modbus", DomainID: "factory-01", Region: "power-room", ModelID: "model-power-meter", Token: generateToken()},
		{ID: "meter_002", Name: "车间A电表", DeviceType: "sensor", Protocol: "modbus", DomainID: "factory-01", Region: "workshop-a", ModelID: "model-power-meter", Token: generateToken()},
		{ID: "opc_001", Name: "OPC UA 设备1号", DeviceType: "plc", Protocol: "opcua", DomainID: "factory-01", Region: "workshop-a", ModelID: "", Token: generateToken()},
		// factory-02 设备
		{ID: "th_sensor_101", Name: "二号工厂温湿度传感器", DeviceType: "sensor", Protocol: "mqtt", DomainID: "factory-02", Region: "main", ModelID: "", Token: generateToken()},
		{ID: "gas_sensor_001", Name: "车间A气体检测仪1号", DeviceType: "sensor", Protocol: "mqtt", DomainID: "factory-02", Region: "workshop-a", ModelID: "model-gas-sensor", Token: generateToken()},
		{ID: "gas_sensor_002", Name: "车间B气体检测仪1号", DeviceType: "sensor", Protocol: "mqtt", DomainID: "factory-02", Region: "workshop-b", ModelID: "model-gas-sensor", Token: generateToken()},
		{ID: "water_meter_001", Name: "厂区总水表", DeviceType: "sensor", Protocol: "modbus", DomainID: "factory-02", Region: "main", ModelID: "model-water-meter", Token: generateToken()},
		{ID: "water_meter_002", Name: "车间A水表", DeviceType: "sensor", Protocol: "modbus", DomainID: "factory-02", Region: "workshop-a", ModelID: "model-water-meter", Token: generateToken()},
	}
	for _, d := range devices {
		db.Create(&d)
	}

	// 3b. 设备-模型绑定
	bindings := []model.DeviceModelBinding{
		// factory-01
		{DeviceID: "th_sensor_001", ModelID: "model-thermometer"},
		{DeviceID: "th_sensor_002", ModelID: "model-thermometer"},
		{DeviceID: "th_sensor_003", ModelID: "model-thermometer"},
		{DeviceID: "motor_001", ModelID: "model-motor"},
		{DeviceID: "motor_002", ModelID: "model-motor"},
		{DeviceID: "motor_003", ModelID: "model-motor"},
		{DeviceID: "meter_001", ModelID: "model-power-meter"},
		{DeviceID: "meter_002", ModelID: "model-power-meter"},
		// factory-02
		{DeviceID: "gas_sensor_001", ModelID: "model-gas-sensor"},
		{DeviceID: "gas_sensor_002", ModelID: "model-gas-sensor"},
		{DeviceID: "water_meter_001", ModelID: "model-water-meter"},
		{DeviceID: "water_meter_002", ModelID: "model-water-meter"},
	}
	for _, b := range bindings {
		db.Create(&b)
	}

	// 4. 规则
	rules := []rule.RuleConfig{
		// factory-01 规则
		{
			ID: "rule-temp-alert", Name: "温度告警规则", DomainID: "factory-01", Topic: "sensor/temperature", Enabled: true,
			Chain: `[{"id":"filter-temp","type":"filter","config":{"field":"topic","operator":"prefix","value":"sensor/"}},{"id":"cond-high","type":"condition","config":{"expression":"temp > 35","true_branch":"","false_branch":""}},{"id":"action-alert","type":"action","config":{"type":"publish","topic_template":"alerts/temperature","payload_template":{"alert":"high_temperature","severity":"warning"}}}]`,
		},
		{
			ID: "rule-motor-monitor", Name: "电机振动监控", DomainID: "factory-01", Topic: "sensor/motor/#", Enabled: true,
			Chain: `[{"id":"filter-motor","type":"filter","config":{"field":"topic","operator":"prefix","value":"sensor/motor/"}},{"id":"agg-vibration","type":"aggregate","config":{"window_size":10,"function":"avg"}},{"id":"cond-vibration","type":"condition","config":{"expression":"vibration > 30","true_branch":"","false_branch":""}},{"id":"action-vibration","type":"action","config":{"type":"publish","topic_template":"alerts/vibration","payload_template":{"alert":"high_vibration","severity":"critical"}}}]`,
		},
		// factory-02 规则
		{
			ID: "rule-gas-leak", Name: "气体泄漏告警", DomainID: "factory-02", Topic: "sensor/gas/#", Enabled: true,
			Chain: `[{"id":"filter-gas","type":"filter","config":{"field":"topic","operator":"prefix","value":"sensor/gas/"}},{"id":"cond-co2","type":"condition","config":{"expression":"co2 > 1000","true_branch":"","false_branch":""}},{"id":"action-gas","type":"action","config":{"type":"alert","topic_template":"system.alerts.factory-02","payload_template":{"alert":"gas_concentration_high","severity":"critical"}}}]`,
		},
		{
			ID: "rule-water-leak", Name: "水管泄漏监控", DomainID: "factory-02", Topic: "sensor/water/#", Enabled: true,
			Chain: `[{"id":"filter-water","type":"filter","config":{"field":"topic","operator":"prefix","value":"sensor/water/"}},{"id":"cond-flow","type":"condition","config":{"expression":"flow_rate > 80","true_branch":"","false_branch":""}},{"id":"action-water","type":"action","config":{"type":"alert","topic_template":"system.alerts.factory-02","payload_template":{"alert":"abnormal_water_flow","severity":"warning"}}}]`,
		},
	}
	for _, r := range rules {
		db.Create(&r)
	}

	// 5. Webhook
	webhooks := []alert.WebhookConfig{
		// factory-01 Webhook
		{
			Name: "钉钉告警通知", DomainID: "factory-01", URL: "https://oapi.dingtalk.com/robot/send?access_token=xxx",
			Method: "POST",
			Filter: alert.WebhookFilter{Levels: []alert.AlertLevel{alert.LevelCritical, alert.LevelWarning}, DeviceTypes: []string{}},
			RateLimit: alert.RateLimitConfig{MaxPerMinute: 10, DedupWindow: 300},
		},
		{
			Name: "企业微信通知", DomainID: "factory-01", URL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx",
			Method: "POST",
			Filter: alert.WebhookFilter{Levels: []alert.AlertLevel{alert.LevelCritical}, DeviceTypes: []string{}},
			RateLimit: alert.RateLimitConfig{MaxPerMinute: 5, DedupWindow: 600},
		},
		// factory-02 Webhook
		{
			Name: "二号工厂钉钉通知", DomainID: "factory-02", URL: "https://oapi.dingtalk.com/robot/send?access_token=yyy",
			Method: "POST",
			Filter: alert.WebhookFilter{Levels: []alert.AlertLevel{alert.LevelCritical, alert.LevelWarning, alert.LevelInfo}, DeviceTypes: []string{}},
			RateLimit: alert.RateLimitConfig{MaxPerMinute: 20, DedupWindow: 300},
		},
	}
	for _, w := range webhooks {
		db.Create(&w)
	}

	// 6. 网关配置
	mqttCfg, _ := json.Marshal(gateway.MQTTConfig{Port: 1884, MaxConnection: 200, KeepAlive: 60})
	tcpCfg, _ := json.Marshal(gateway.TCPConfig{Port: 9000, MaxConnection: 100, Heartbeat: 30})
	modbusCfg, _ := json.Marshal(gateway.ModbusConfig{Port: 502, PollInterval: 10, SlaveIDs: []int{1, 2, 3}})
	opcuaCfg, _ := json.Marshal(gateway.OPCUAConfig{Endpoint: "opc.tcp://localhost:4840", PollInterval: 5, NodeIDs: []string{"ns=2;s=Temperature", "ns=2;s=Humidity"}, DeviceID: "opc_001", DomainID: "factory-01"})

	gateways := []gateway.GatewayConfig{
		{ID: "gw_mqtt_001", Name: "MQTT 主网关", Type: gateway.TypeMQTT, Config: string(mqttCfg), Enabled: true, Status: gateway.StatusStopped},
		{ID: "gw_tcp_001", Name: "TCP 工业网关", Type: gateway.TypeTCP, Config: string(tcpCfg), Enabled: true, Status: gateway.StatusStopped},
		{ID: "gw_modbus_001", Name: "Modbus 网关", Type: gateway.TypeModbus, Config: string(modbusCfg), Enabled: true, Status: gateway.StatusStopped},
		{ID: "gw_opcua_001", Name: "OPC UA 网关", Type: gateway.TypeOPCUA, Config: string(opcuaCfg), Enabled: true, Status: gateway.StatusStopped},
	}
	for _, g := range gateways {
		db.Create(&g)
	}

	log.Printf("种子数据已填充: %d 个域, %d 个物模型, %d 个设备, %d 个绑定, %d 条规则, %d 个 Webhook, %d 个网关",
		len(domains), len(models), len(devices), len(bindings), len(rules), len(webhooks), len(gateways))
}
