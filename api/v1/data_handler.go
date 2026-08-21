package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/datacenter/internal/timeseries"
	"github.com/gin-gonic/gin"
)

// DataHandler 设备时序数据 API
type DataHandler struct {
	queryService *timeseries.QueryService
}

// NewDataHandler 创建数据处理器
func NewDataHandler(queryService *timeseries.QueryService) *DataHandler {
	return &DataHandler{queryService: queryService}
}

// RegisterRoutes 注册路由
func (h *DataHandler) RegisterRoutes(r *gin.RouterGroup) {
	data := r.Group("/data")
	{
		data.GET("/device/:id", h.GetDeviceData)
		data.GET("/device/:id/latest", h.GetLatestData)
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
