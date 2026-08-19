package rule

import (
	"encoding/json"
	"time"
)

// RuleConfig 规则持久化模型（GORM）
type RuleConfig struct {
	ID        string    `json:"id" gorm:"primaryKey;size:64"`
	Name      string    `json:"name" gorm:"size:128"`
	DomainID  string    `json:"domain_id" gorm:"size:64;index"`
	Topic     string    `json:"topic" gorm:"size:256"`
	Chain     string    `json:"chain" gorm:"type:text"` // JSON 编码的 []NodeConfig
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name     string       `json:"name" binding:"required"`
	DomainID string       `json:"domain_id" binding:"required"`
	Topic    string       `json:"topic"`
	Chain    []NodeConfig `json:"chain" binding:"required"`
	Enabled  bool         `json:"enabled"`
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name     *string      `json:"name"`
	Topic    *string      `json:"topic"`
	Chain    []NodeConfig `json:"chain"`
	Enabled  *bool        `json:"enabled"`
}

// toRuleConfig 将 Rule 和 domainID 转换为 RuleConfig
func toRuleConfig(r *Rule, domainID string) (*RuleConfig, error) {
	chainJSON, err := json.Marshal(r.Chain)
	if err != nil {
		return nil, err
	}
	return &RuleConfig{
		ID:        r.ID,
		Name:      r.Name,
		DomainID:  domainID,
		Topic:     r.Topic,
		Chain:     string(chainJSON),
		Enabled:   r.Enabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// toRule 将 RuleConfig 转换为 Rule
func toRule(rc *RuleConfig) (*Rule, error) {
	var chain []NodeConfig
	if err := json.Unmarshal([]byte(rc.Chain), &chain); err != nil {
		return nil, err
	}
	return &Rule{
		ID:       rc.ID,
		Name:     rc.Name,
		DomainID: rc.DomainID,
		Topic:    rc.Topic,
		Chain:    chain,
		Enabled:  rc.Enabled,
	}, nil
}
