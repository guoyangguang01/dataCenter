package v1

import (
	"net/http"

	"github.com/datacenter/internal/rule"
	"github.com/gin-gonic/gin"
)

type RuleHandler struct {
	service *rule.RuleConfigService
}

func NewRuleHandler(service *rule.RuleConfigService) *RuleHandler {
	return &RuleHandler{service: service}
}

func (h *RuleHandler) RegisterRoutes(r *gin.RouterGroup) {
	rules := r.Group("/rules")
	{
		rules.POST("", h.Create)
		rules.GET("", h.List)
		rules.GET("/:id", h.GetByID)
		rules.PUT("/:id", h.Update)
		rules.DELETE("/:id", h.Delete)
		rules.PUT("/:id/toggle", h.ToggleEnabled)
	}
}

func (h *RuleHandler) Create(c *gin.Context) {
	var req rule.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rc, err := h.service.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rc)
}

func (h *RuleHandler) List(c *gin.Context) {
	domainID := c.Query("domain_id")
	configs, err := h.service.ListByDomain(domainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (h *RuleHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	rc, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}
	c.JSON(http.StatusOK, rc)
}

func (h *RuleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req rule.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rc, err := h.service.Update(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rc)
}

func (h *RuleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *RuleHandler) ToggleEnabled(c *gin.Context) {
	id := c.Param("id")
	rc, err := h.service.ToggleEnabled(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rc)
}
