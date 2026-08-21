package v1

import (
	"net/http"

	"github.com/datacenter/internal/alert"
	"github.com/datacenter/internal/device"
	"github.com/datacenter/internal/gateway"
	"github.com/gin-gonic/gin"
)

// StatsHandler 统计与监控 API
type StatsHandler struct {
	deviceService  *device.Service
	alertService   *alert.AlertService
	gatewayService *gateway.GatewayService
	launcher       *gateway.Launcher
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(
	deviceService *device.Service,
	alertService *alert.AlertService,
	gatewayService *gateway.GatewayService,
	launcher *gateway.Launcher,
) *StatsHandler {
	return &StatsHandler{
		deviceService:  deviceService,
		alertService:   alertService,
		gatewayService: gatewayService,
		launcher:       launcher,
	}
}

// RegisterRoutes 注册路由
func (h *StatsHandler) RegisterRoutes(r *gin.RouterGroup) {
	stats := r.Group("/stats")
	{
		stats.GET("/dashboard", h.Dashboard)
		stats.GET("/monitoring", h.Monitoring)
	}
}

// Dashboard 返回仪表盘聚合数据
func (h *StatsHandler) Dashboard(c *gin.Context) {
	deviceTotal, _ := h.deviceService.Count()
	deviceOnline, _ := h.deviceService.CountOnline()
	alertCount, _ := h.alertService.CountRecent()
	recentAlerts, _ := h.alertService.ListAlertLogs("")

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_total":   deviceTotal,
			"device_online":  deviceOnline,
			"message_today":  0, // 消息计数来自 timeseries-writer，跨进程不可用
			"alert_count":    alertCount,
			"recent_alerts":  recentAlerts,
		},
	})
}

// Monitoring 返回系统监控数据
func (h *StatsHandler) Monitoring(c *gin.Context) {
	// 网关状态
	configs, _ := h.gatewayService.List()
	runningCount := h.launcher.RunningCount()

	// 设备统计
	deviceTotal, _ := h.deviceService.Count()
	deviceOnline, _ := h.deviceService.CountOnline()

	// 构造网关列表
	gateways := make([]gin.H, 0, len(configs))
	for _, gc := range configs {
		gateways = append(gateways, gin.H{
			"id":     gc.ID,
			"name":   gc.Name,
			"type":   gc.Type,
			"status": gc.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"gateways":       gateways,
			"gateway_running": runningCount,
			"gateway_total":   len(configs),
			"device_online":   deviceOnline,
			"device_total":    deviceTotal,
		},
	})
}
