package v1

import (
	"net/http"

	"github.com/datacenter/internal/gateway"
	"github.com/gin-gonic/gin"
)

type GatewayHandler struct {
	service  *gateway.GatewayService
	launcher *gateway.Launcher
}

func NewGatewayHandler(service *gateway.GatewayService, launcher *gateway.Launcher) *GatewayHandler {
	return &GatewayHandler{service: service, launcher: launcher}
}

func (h *GatewayHandler) RegisterRoutes(r *gin.RouterGroup) {
	gateways := r.Group("/gateways")
	{
		gateways.POST("", h.Create)
		gateways.GET("", h.List)
		gateways.GET("/:id", h.GetByID)
		gateways.PUT("/:id", h.Update)
		gateways.DELETE("/:id", h.Delete)
		gateways.POST("/:id/start", h.Start)
		gateways.POST("/:id/stop", h.Stop)
	}
}

func (h *GatewayHandler) Create(c *gin.Context) {
	var req gateway.CreateGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gc, err := h.service.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gc)
}

func (h *GatewayHandler) List(c *gin.Context) {
	configs, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (h *GatewayHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	gc, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "gateway not found"})
		return
	}
	c.JSON(http.StatusOK, gc)
}

func (h *GatewayHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req gateway.CreateGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gc, err := h.service.Update(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gc)
}

func (h *GatewayHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	// 先停止运行中的网关
	h.launcher.StopGateway(id)
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *GatewayHandler) Start(c *gin.Context) {
	id := c.Param("id")
	if err := h.launcher.StartGateway(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "started"})
}

func (h *GatewayHandler) Stop(c *gin.Context) {
	id := c.Param("id")
	if err := h.launcher.StopGateway(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stopped"})
}
