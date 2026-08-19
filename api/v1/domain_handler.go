package v1

import (
	"net/http"

	"github.com/datacenter/internal/domain"
	"github.com/gin-gonic/gin"
)

type DomainHandler struct {
	service *domain.Service
}

func NewDomainHandler(service *domain.Service) *DomainHandler {
	return &DomainHandler{service: service}
}

func (h *DomainHandler) RegisterRoutes(r *gin.RouterGroup) {
	domains := r.Group("/domains")
	{
		domains.POST("", h.Create)
		domains.GET("", h.List)
		domains.GET("/:id", h.GetByID)
		domains.DELETE("/:id", h.Delete)
		domains.POST("/:id/members", h.AddMember)
		domains.GET("/:id/members", h.ListMembers)
		domains.DELETE("/:id/members/:userId", h.RemoveMember)
	}
}

func (h *DomainHandler) Create(c *gin.Context) {
	var req domain.CreateDomainRequest
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

func (h *DomainHandler) List(c *gin.Context) {
	domains, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": domains})
}

func (h *DomainHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	d, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *DomainHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *DomainHandler) AddMember(c *gin.Context) {
	domainID := c.Param("id")
	var req domain.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddMember(domainID, req.UserID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "member added"})
}

func (h *DomainHandler) ListMembers(c *gin.Context) {
	domainID := c.Param("id")
	members, err := h.service.ListMembers(domainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": members})
}

func (h *DomainHandler) RemoveMember(c *gin.Context) {
	domainID := c.Param("id")
	userID := c.Param("userId")
	if err := h.service.RemoveMember(domainID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}
