package v1

import (
	"net/http"

	"github.com/datacenter/internal/gateway"
	"github.com/gin-gonic/gin"
)

// CommandHandler 设备命令下发 API
type CommandHandler struct {
	publisher gateway.CommandPublisher
}

// NewCommandHandler 创建命令处理器
func NewCommandHandler(publisher gateway.CommandPublisher) *CommandHandler {
	return &CommandHandler{publisher: publisher}
}

// RegisterRoutes 注册路由
func (h *CommandHandler) RegisterRoutes(r *gin.RouterGroup) {
	devices := r.Group("/devices")
	{
		devices.POST("/:id/command", h.SendCommand)
	}
}

// SendCommandRequest 命令下发请求
type SendCommandRequest struct {
	Payload []byte `json:"payload" binding:"required"`
}

// SendCommand 向指定设备发送下行命令
func (h *CommandHandler) SendCommand(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device id is required"})
		return
	}

	var req SendCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.publisher.SendToDevice(deviceID, req.Payload); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "sent",
		"device_id": deviceID,
	})
}
