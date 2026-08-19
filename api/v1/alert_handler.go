package v1

import (
	"net/http"

	"github.com/datacenter/internal/alert"
	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	service *alert.AlertService
}

func NewAlertHandler(service *alert.AlertService) *AlertHandler {
	return &AlertHandler{service: service}
}

func (h *AlertHandler) RegisterRoutes(r *gin.RouterGroup) {
	alerts := r.Group("/alerts")
	{
		alerts.POST("/webhooks", h.CreateWebhook)
		alerts.GET("/webhooks", h.ListWebhooks)
		alerts.GET("/webhooks/:id", h.GetWebhook)
		alerts.PUT("/webhooks/:id", h.UpdateWebhook)
		alerts.DELETE("/webhooks/:id", h.DeleteWebhook)
		alerts.POST("/webhooks/:id/test", h.TestWebhook)
		alerts.GET("/logs", h.ListAlertLogs)
	}
}

func (h *AlertHandler) CreateWebhook(c *gin.Context) {
	var req alert.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateWebhook(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

func (h *AlertHandler) ListWebhooks(c *gin.Context) {
	domainID := c.Query("domain_id")
	configs, err := h.service.ListWebhooksByDomain(domainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (h *AlertHandler) GetWebhook(c *gin.Context) {
	id := c.Param("id")
	config, err := h.service.GetWebhook(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *AlertHandler) UpdateWebhook(c *gin.Context) {
	id := c.Param("id")
	var req alert.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.UpdateWebhook(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *AlertHandler) DeleteWebhook(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteWebhook(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *AlertHandler) TestWebhook(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.TestWebhook(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "test sent"})
}

func (h *AlertHandler) ListAlertLogs(c *gin.Context) {
	webhookID := c.Query("webhook_id")
	logs, err := h.service.ListAlertLogs(webhookID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}
