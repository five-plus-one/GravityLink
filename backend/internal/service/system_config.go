package service

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gravitylink/backend/internal/model"
)

type SystemConfigService struct {
	db *gorm.DB
}

type SystemConfigInput struct {
	Configs map[string]string `json:"configs"`
}

func NewSystemConfigService(db *gorm.DB) *SystemConfigService {
	return &SystemConfigService{db: db}
}

func (s *SystemConfigService) List(ctx context.Context) (map[string]string, error) {
	var items []model.SystemConfig
	if err := s.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	configs := make(map[string]string, len(items))
	for _, item := range items {
		configs[item.KeyName] = item.Value
	}
	return configs, nil
}

func (s *SystemConfigService) Update(ctx context.Context, input SystemConfigInput) (map[string]string, error) {
	for key, value := range input.Configs {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		item := model.SystemConfig{KeyName: key, Value: value}
		err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).Create(&item).Error
		if err != nil {
			return nil, err
		}
	}
	return s.List(ctx)
}
