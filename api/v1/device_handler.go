package v1

import (
	"net/http"
	"strconv"

	"github.com/datacenter/internal/device"
	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	service *device.Service
}

func NewDeviceHandler(service *device.Service) *DeviceHandler {
	return &DeviceHandler{service: service}
}

func (h *DeviceHandler) RegisterRoutes(r *gin.RouterGroup) {
	devices := r.Group("/devices")
	{
		devices.POST("", h.Create)
		devices.GET("", h.List)
		devices.GET("/:id", h.GetByID)
		devices.PUT("/:id", h.Update)
		devices.DELETE("/:id", h.Delete)
		devices.POST("/:id/verify", h.Verify)
	}
}

func (h *DeviceHandler) Create(c *gin.Context) {
	var req device.CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	d, err := h.service.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, d)
}

func (h *DeviceHandler) List(c *gin.Context) {
	var query device.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	devices, total, err := h.service.List(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  devices,
		"total": total,
		"page":  query.Page,
		"size":  query.PageSize,
	})
}

func (h *DeviceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	d, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *DeviceHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req device.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	d, err := h.service.Update(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, d)
}

func (h *DeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *DeviceHandler) Verify(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	d, err := h.service.VerifyToken(id, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"device": d, "verified": true})
}

func parsePage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}
