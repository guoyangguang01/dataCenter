package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/datacenter/internal/model"
	"github.com/datacenter/internal/timeseries"
	"github.com/gin-gonic/gin"
)

// MetricGroup 按指标分组的最新数据
type MetricGroup struct {
	Name    string          `json:"name"`
	Unit    string          `json:"unit"`
	Devices []MetricDevice  `json:"devices"`
}

// MetricDevice 单个设备的指标值
type MetricDevice struct {
	DeviceID   string  `json:"device_id"`
	DeviceName string  `json:"device_name"`
	Value      float64 `json:"value"`
	TS         string  `json:"ts"`
	Quality    int     `json:"quality"`
}

// DataHandler 设备时序数据 API
type DataHandler struct {
	queryService *timeseries.QueryService
	modelService *model.Service
}

// NewDataHandler 创建数据处理器
func NewDataHandler(queryService *timeseries.QueryService, modelService *model.Service) *DataHandler {
	return &DataHandler{queryService: queryService, modelService: modelService}
}

// RegisterRoutes 注册路由
func (h *DataHandler) RegisterRoutes(r *gin.RouterGroup) {
	data := r.Group("/data")
	{
		data.GET("/device/:id", h.GetDeviceData)
		data.GET("/device/:id/latest", h.GetLatestData)
		data.GET("/metrics", h.GetMetrics)
		data.GET("/metric/:metric/latest", h.GetMetricLatest)
		data.GET("/metric/:metric/history", h.GetMetricHistory)
		data.GET("/all-latest", h.GetAllLatest)
	}
}

// GetDeviceData 查询设备历史数据
func (h *DataHandler) GetDeviceData(c *gin.Context) {
	deviceID := c.Param("id")
	hoursStr := c.DefaultQuery("hours", "1")
	hours, _ := strconv.Atoi(hoursStr)
	if hours <= 0 {
		hours = 1
	}

	start := time.Now().Add(-time.Duration(hours) * time.Hour)
	end := time.Now()

	result, err := h.queryService.QueryByDevice(deviceID, start, end)
	if err != nil {
		// TDengine 不可用时返回空数据，不报错
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"device_id": deviceID,
				"columns":   []string{"ts", "value", "quality"},
				"rows":      [][]interface{}{},
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id":    deviceID,
			"columns":      result.ColumnNames,
			"rows":         result.Rows,
			"rows_affected": result.RowsAffected,
		},
	})
}

// GetLatestData 查询设备最新数据
func (h *DataHandler) GetLatestData(c *gin.Context) {
	deviceID := c.Param("id")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	}

	result, err := h.queryService.QueryLatest(deviceID, limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"device_id": deviceID,
				"columns":   []string{},
				"rows":      [][]interface{}{},
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id":    deviceID,
			"columns":      result.ColumnNames,
			"rows":         result.Rows,
			"rows_affected": result.RowsAffected,
		},
	})
}

// GetMetrics 获取所有可用指标列表（从物模型提取）
func (h *DataHandler) GetMetrics(c *gin.Context) {
	if h.modelService == nil {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}

	models, err := h.modelService.ListAll()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}

	// 去重收集所有指标
	seen := make(map[string]bool)
	metrics := make([]gin.H, 0)
	for _, m := range models {
		for _, p := range m.Properties {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			metrics = append(metrics, gin.H{
				"id":    p.ID,
				"name":  p.Name,
				"unit":  p.Unit,
				"type":  p.DataType,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

// GetMetricLatest 跨设备查询指定指标的最新值
func (h *DataHandler) GetMetricLatest(c *gin.Context) {
	metric := c.Param("metric")
	domainID := c.Query("domain_id")
	if h.queryService == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"metric": metric, "devices": []interface{}{}}})
		return
	}

	result, err := h.queryService.QueryLatestMetric(metric, domainID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"metric": metric, "devices": []interface{}{}}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetMetricHistory 跨设备查询指定指标的历史趋势
func (h *DataHandler) GetMetricHistory(c *gin.Context) {
	metric := c.Param("metric")
	domainID := c.Query("domain_id")
	hoursStr := c.DefaultQuery("hours", "1")
	hours, _ := strconv.Atoi(hoursStr)
	if hours <= 0 {
		hours = 1
	}

	start := time.Now().Add(-time.Duration(hours) * time.Hour)
	end := time.Now()

	if h.queryService == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"metric": metric, "devices": []interface{}{}}})
		return
	}

	result, err := h.queryService.QueryMetricHistory(metric, domainID, start, end)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"metric": metric, "devices": []interface{}{}}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetAllLatest 查询所有设备最新数据，按指标分组返回
func (h *DataHandler) GetAllLatest(c *gin.Context) {
	domainID := c.Query("domain_id")
	if h.queryService == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{}})
		return
	}

	// 查询所有设备最新数据
	result, err := h.queryService.QueryLatestAll(domainID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{}})
		return
	}

	// 构建指标元数据映射（从物模型获取 name/unit）
	metricMeta := make(map[string]struct{ Name, Unit string })
	if h.modelService != nil {
		if models, err := h.modelService.ListAll(); err == nil {
			for _, m := range models {
				for _, p := range m.Properties {
					if _, exists := metricMeta[p.ID]; !exists {
						metricMeta[p.ID] = struct{ Name, Unit string }{p.Name, p.Unit}
					}
				}
			}
		}
	}

	// 按指标分组
	grouped := make(map[string]*MetricGroup)
	for _, row := range result.Rows {
		if len(row) < 4 {
			continue
		}
		deviceID, _ := row[0].(string)
		ts := fmt.Sprintf("%v", row[1])
		valStr := fmt.Sprintf("%v", row[2])
		quality := 0
		if q, ok := row[3].(int64); ok {
			quality = int(q)
		}

		// 解析 val JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(valStr), &parsed); err != nil {
			continue
		}

		realDeviceID := strings.ReplaceAll(deviceID, "_", "-")

		for key, v := range parsed {
			if key == "device_id" || key == "timestamp" {
				continue
			}
			var numVal float64
			switch n := v.(type) {
			case float64:
				numVal = n
			case json.Number:
				numVal, _ = n.Float64()
			default:
				continue
			}

			if _, exists := grouped[key]; !exists {
				meta := metricMeta[key]
				name := meta.Name
				if name == "" {
					name = key
				}
				grouped[key] = &MetricGroup{Name: name, Unit: meta.Unit, Devices: make([]MetricDevice, 0)}
			}
			grouped[key].Devices = append(grouped[key].Devices, MetricDevice{
				DeviceID: realDeviceID,
				Value:    numVal,
				TS:       ts,
				Quality:  quality,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": grouped})
}
