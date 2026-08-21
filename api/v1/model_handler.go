package v1

import (
	"net/http"

	"github.com/datacenter/internal/model"
	"github.com/gin-gonic/gin"
)

type ModelHandler struct {
	service *model.Service
}

func NewModelHandler(service *model.Service) *ModelHandler {
	return &ModelHandler{service: service}
}

func (h *ModelHandler) RegisterRoutes(r *gin.RouterGroup) {
	models := r.Group("/models")
	{
		models.POST("", h.Create)
		models.GET("", h.List)
		models.GET("/:id", h.GetByID)
		models.DELETE("/:id", h.Delete)
		models.POST("/bind", h.BindDevice)
		models.DELETE("/unbind/:deviceId", h.UnbindDevice)
		models.GET("/device/:deviceId", h.GetDeviceModel)
	}
}

func (h *ModelHandler) Create(c *gin.Context) {
	var req model.CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m, err := h.service.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, m)
}

func (h *ModelHandler) List(c *gin.Context) {
	domainID := c.Query("domain_id")

	var models []model.ThingModel
	var err error
	if domainID == "" {
		models, err = h.service.ListAll()
	} else {
		models, err = h.service.ListByDomain(domainID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": models})
}

func (h *ModelHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	m, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *ModelHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *ModelHandler) BindDevice(c *gin.Context) {
	var req model.BindModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.BindDevice(req.DeviceID, req.ModelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bound"})
}

func (h *ModelHandler) UnbindDevice(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if err := h.service.UnbindDevice(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unbound"})
}

func (h *ModelHandler) GetDeviceModel(c *gin.Context) {
	deviceID := c.Param("deviceId")
	m, err := h.service.GetDeviceModel(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no model bound to this device"})
		return
	}
	c.JSON(http.StatusOK, m)
}
