package domain

import (
	"time"
)

type DomainRole string

const (
	RoleSuperAdmin DomainRole = "super_admin"
	RoleAdmin      DomainRole = "admin"
	RoleOperator   DomainRole = "operator"
	RoleViewer     DomainRole = "viewer"
)

type Domain struct {
	ID          string    `json:"id" gorm:"primaryKey;size:64"`
	Name        string    `json:"name" gorm:"size:128"`
	Description string    `json:"description" gorm:"size:512"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DomainMember struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	DomainID  string     `json:"domain_id" gorm:"size:64;index"`
	UserID    string     `json:"user_id" gorm:"size:64;index"`
	Role      DomainRole `json:"role" gorm:"size:16"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateDomainRequest struct {
	ID          string `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type AddMemberRequest struct {
	UserID string     `json:"user_id" binding:"required"`
	Role   DomainRole `json:"role" binding:"required"`
}
