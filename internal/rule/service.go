package rule

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// RuleConfigService 规则持久化服务
type RuleConfigService struct {
	db     *gorm.DB
	engine *Engine
}

// NewRuleConfigService 创建规则持久化服务
func NewRuleConfigService(db *gorm.DB, engine *Engine) *RuleConfigService {
	return &RuleConfigService{db: db, engine: engine}
}

// InitDB 初始化数据库表
func (s *RuleConfigService) InitDB() error {
	return s.db.AutoMigrate(&RuleConfig{})
}

// LoadAll 从数据库加载所有启用规则到引擎
func (s *RuleConfigService) LoadAll() error {
	var configs []RuleConfig
	if err := s.db.Where("enabled = ?", true).Find(&configs).Error; err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	for _, rc := range configs {
		rule, err := toRule(&rc)
		if err != nil {
			fmt.Printf("[Rule] failed to convert rule %s: %v\n", rc.ID, err)
			continue
		}
		if err := s.engine.AddRule(*rule); err != nil {
			fmt.Printf("[Rule] failed to load rule %s: %v\n", rc.ID, err)
			continue
		}
	}

	fmt.Printf("[Rule] loaded %d rules from database\n", len(configs))
	return nil
}

// Create 创建规则
func (s *RuleConfigService) Create(req CreateRuleRequest) (*RuleConfig, error) {
	rule := &Rule{
		ID:       req.Name, // 使用 Name 作为 ID（简化），或自动生成
		Name:     req.Name,
		DomainID: req.DomainID,
		Topic:    req.Topic,
		Chain:    req.Chain,
		Enabled:  req.Enabled,
	}

	// 生成唯一 ID
	rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())

	rc, err := toRuleConfig(rule, req.DomainID)
	if err != nil {
		return nil, err
	}

	if err := s.db.Create(rc).Error; err != nil {
		return nil, err
	}

	// 如果启用，添加到引擎
	if req.Enabled {
		if err := s.engine.AddRule(*rule); err != nil {
			fmt.Printf("[Rule] failed to add rule to engine: %v\n", err)
		}
	}

	return rc, nil
}

// GetByID 获取规则
func (s *RuleConfigService) GetByID(id string) (*RuleConfig, error) {
	var rc RuleConfig
	if err := s.db.Where("id = ?", id).First(&rc).Error; err != nil {
		return nil, err
	}
	return &rc, nil
}

// ListByDomain 按域查询规则
func (s *RuleConfigService) ListByDomain(domainID string) ([]RuleConfig, error) {
	var configs []RuleConfig
	query := s.db
	if domainID != "" {
		query = query.Where("domain_id = ?", domainID)
	}
	if err := query.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// Update 更新规则
func (s *RuleConfigService) Update(id string, req UpdateRuleRequest) (*RuleConfig, error) {
	rc, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Topic != nil {
		updates["topic"] = *req.Topic
	}
	if req.Chain != nil {
		chainJSON, err := marshalChain(req.Chain)
		if err != nil {
			return nil, err
		}
		updates["chain"] = chainJSON
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	updates["updated_at"] = time.Now()

	if err := s.db.Model(rc).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 同步引擎：先移除再添加
	s.engine.RemoveRule(id)

	if rc.Enabled {
		rule, err := toRule(rc)
		if err != nil {
			return nil, err
		}
		// 应用更新到 rule
		if req.Name != nil {
			rule.Name = *req.Name
		}
		if req.Topic != nil {
			rule.Topic = *req.Topic
		}
		if req.Chain != nil {
			rule.Chain = req.Chain
		}
		if req.Enabled != nil {
			rule.Enabled = *req.Enabled
		}
		if rule.Enabled {
			if err := s.engine.AddRule(*rule); err != nil {
				fmt.Printf("[Rule] failed to update rule in engine: %v\n", err)
			}
		}
	}

	// 重新获取更新后的记录
	return s.GetByID(id)
}

// Delete 删除规则
func (s *RuleConfigService) Delete(id string) error {
	if err := s.db.Where("id = ?", id).Delete(&RuleConfig{}).Error; err != nil {
		return err
	}
	s.engine.RemoveRule(id)
	return nil
}

// ToggleEnabled 切换启用状态
func (s *RuleConfigService) ToggleEnabled(id string) (*RuleConfig, error) {
	rc, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	newEnabled := !rc.Enabled
	if err := s.db.Model(rc).Updates(map[string]interface{}{
		"enabled":    newEnabled,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, err
	}

	if newEnabled {
		rule, err := toRule(rc)
		if err != nil {
			return nil, err
		}
		rule.Enabled = true
		if err := s.engine.AddRule(*rule); err != nil {
			fmt.Printf("[Rule] failed to add rule to engine: %v\n", err)
		}
	} else {
		s.engine.RemoveRule(id)
	}

	rc.Enabled = newEnabled
	return rc, nil
}

// marshalChain 序列化节点链
func marshalChain(chain []NodeConfig) (string, error) {
	data, err := json.Marshal(chain)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
