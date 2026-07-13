package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

var ErrNoRoutingTarget = errors.New("no routing target available")

type RoutingService struct {
	db    *gorm.DB
	redis *redis.Client
}

type StrategyInput struct {
	Mode    string               `json:"mode"`
	Targets []RoutingTargetInput `json:"targets"`
}

type RoutingTargetInput struct {
	Label     string `json:"label"`
	TargetURL string `json:"target_url"`
	Weight    uint   `json:"weight"`
	ScanLimit *uint  `json:"scan_limit"`
}

func NewRoutingService(db *gorm.DB, redis *redis.Client) *RoutingService {
	return &RoutingService{db: db, redis: redis}
}

func (s *RoutingService) CreateStrategy(tx *gorm.DB, linkID uint64, input *StrategyInput) error {
	if input == nil {
		return ErrNoRoutingTarget
	}
	mode := input.Mode
	if mode == "" {
		mode = "round_robin"
	}
	if mode != "round_robin" && mode != "weighted" {
		return ErrUnsupportedLink
	}
	if len(input.Targets) == 0 {
		return ErrNoRoutingTarget
	}

	strategy := model.RoutingStrategy{LinkID: linkID, Mode: mode}
	if err := tx.Create(&strategy).Error; err != nil {
		return err
	}

	targets := make([]model.RoutingTarget, 0, len(input.Targets))
	for _, item := range input.Targets {
		if item.TargetURL == "" {
			return ErrTargetUnavailable
		}
		weight := item.Weight
		if weight == 0 {
			weight = 1
		}
		targets = append(targets, model.RoutingTarget{
			StrategyID: strategy.ID,
			Label:      optionalString(item.Label),
			TargetURL:  item.TargetURL,
			Weight:     weight,
			ScanLimit:  item.ScanLimit,
			Status:     model.StatusActive,
		})
	}
	return tx.Create(&targets).Error
}

func (s *RoutingService) SelectTarget(ctx context.Context, linkID uint64) (model.RoutingTarget, error) {
	var strategy model.RoutingStrategy
	err := s.db.WithContext(ctx).Where("link_id = ?", linkID).First(&strategy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.RoutingTarget{}, ErrNoRoutingTarget
	}
	if err != nil {
		return model.RoutingTarget{}, err
	}

	var targets []model.RoutingTarget
	if err := s.db.WithContext(ctx).
		Where("strategy_id = ? AND status = ?", strategy.ID, model.StatusActive).
		Order("priority DESC, id ASC").
		Find(&targets).Error; err != nil {
		return model.RoutingTarget{}, err
	}
	targets = s.filterAvailable(ctx, targets)
	if len(targets) == 0 {
		return model.RoutingTarget{}, ErrNoRoutingTarget
	}

	var selected model.RoutingTarget
	if strategy.Mode == "weighted" {
		selected = weightedTarget(targets)
	} else {
		selected = targets[0]
	}
	if err := s.incrementScan(ctx, selected); err != nil {
		return model.RoutingTarget{}, err
	}
	return selected, nil
}

func (s *RoutingService) filterAvailable(ctx context.Context, targets []model.RoutingTarget) []model.RoutingTarget {
	available := make([]model.RoutingTarget, 0, len(targets))
	for _, target := range targets {
		if target.ScanLimit == nil {
			available = append(available, target)
			continue
		}
		count, err := s.redis.Get(ctx, routingScanKey(target.ID)).Uint64()
		if errors.Is(err, redis.Nil) {
			count = uint64(target.ScanCount)
		}
		if count < uint64(*target.ScanLimit) {
			available = append(available, target)
		}
	}
	return available
}

func (s *RoutingService) incrementScan(ctx context.Context, target model.RoutingTarget) error {
	count, err := s.redis.Incr(ctx, routingScanKey(target.ID)).Uint64()
	if err != nil {
		return err
	}
	if target.ScanLimit != nil && count > uint64(*target.ScanLimit) {
		_ = s.db.WithContext(ctx).Model(&target).Updates(map[string]interface{}{
			"status":     "exhausted",
			"scan_count": count,
		}).Error
		return ErrNoRoutingTarget
	}
	return s.db.WithContext(ctx).Model(&target).Update("scan_count", count).Error
}

func weightedTarget(targets []model.RoutingTarget) model.RoutingTarget {
	total := uint(0)
	for _, target := range targets {
		total += target.Weight
	}
	if total == 0 {
		return targets[0]
	}
	pick := uint(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(int(total))) + 1
	accumulated := uint(0)
	for _, target := range targets {
		accumulated += target.Weight
		if pick <= accumulated {
			return target
		}
	}
	return targets[0]
}

func routingScanKey(targetID uint64) string {
	return fmt.Sprintf("routing:scan:%d", targetID)
}
