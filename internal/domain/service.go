package domain

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) InitDB() error {
	return s.db.AutoMigrate(&Domain{}, &DomainMember{})
}

func (s *Service) Create(req CreateDomainRequest) (*Domain, error) {
	domain := &Domain{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.db.Create(domain).Error; err != nil {
		return nil, fmt.Errorf("failed to create domain: %w", err)
	}
	return domain, nil
}

func (s *Service) GetByID(id string) (*Domain, error) {
	var domain Domain
	if err := s.db.Where("id = ?", id).First(&domain).Error; err != nil {
		return nil, err
	}
	return &domain, nil
}

func (s *Service) List() ([]Domain, error) {
	var domains []Domain
	err := s.db.Find(&domains).Error
	return domains, err
}

func (s *Service) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&Domain{}).Error
}

func (s *Service) AddMember(domainID, userID string, role DomainRole) error {
	member := &DomainMember{
		DomainID:  domainID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now(),
	}
	return s.db.Create(member).Error
}

func (s *Service) RemoveMember(domainID, userID string) error {
	return s.db.Where("domain_id = ? AND user_id = ?", domainID, userID).Delete(&DomainMember{}).Error
}

func (s *Service) GetMemberRole(domainID, userID string) (DomainRole, error) {
	var member DomainMember
	if err := s.db.Where("domain_id = ? AND user_id = ?", domainID, userID).First(&member).Error; err != nil {
		return "", err
	}
	return member.Role, nil
}

func (s *Service) ListMembers(domainID string) ([]DomainMember, error) {
	var members []DomainMember
	err := s.db.Where("domain_id = ?", domainID).Find(&members).Error
	return members, err
}
