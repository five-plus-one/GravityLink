package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

var ErrNoRoutingTarget = errors.New("no routing target available")

// validRoutingTargetURL 活码二维码目标校验：允许 http(s) 绝对地址或站内素材相对路径。
// 与 router 层 validTargetURL 口径一致，避免同一张图两种存储形态。
func validRoutingTargetURL(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > 2048 {
		return false
	}
	if strings.HasPrefix(trimmed, "/uploads/") && !strings.Contains(trimmed, "..") && !strings.ContainsAny(trimmed, "?#\\") {
		return true
	}
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

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
		// 活码目标允许站内素材相对路径（/uploads/...），与 TargetManager 口径一致
		if !validRoutingTargetURL(item.TargetURL) {
			return ErrInvalidTargetURL
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

	for len(targets) > 0 {
		selected := targets[0]
		if strategy.Mode == "weighted" {
			selected = weightedTarget(targets)
		}
		err := s.incrementScan(ctx, selected)
		if err == nil {
			return selected, nil
		}
		if !errors.Is(err, ErrNoRoutingTarget) {
			return model.RoutingTarget{}, err
		}
		for i := range targets {
			if targets[i].ID == selected.ID {
				targets = append(targets[:i], targets[i+1:]...)
				break
			}
		}
	}
	return model.RoutingTarget{}, ErrNoRoutingTarget
}

func (s *RoutingService) filterAvailable(ctx context.Context, targets []model.RoutingTarget) []model.RoutingTarget {
	available := make([]model.RoutingTarget, 0, len(targets))
	for _, target := range targets {
		if target.ExpireAt != nil && !target.ExpireAt.After(time.Now()) {
			continue
		}
		if target.ScanLimit == nil {
			available = append(available, target)
			continue
		}
		if target.ScanCount < *target.ScanLimit {
			available = append(available, target)
		}
	}
	return available
}

func (s *RoutingService) incrementScan(ctx context.Context, target model.RoutingTarget) error {
	result := s.db.WithContext(ctx).Model(&model.RoutingTarget{}).Where("id = ? AND status = ? AND (expire_at IS NULL OR expire_at > ?) AND (scan_limit IS NULL OR scan_count < scan_limit)", target.ID, model.StatusActive, time.Now()).UpdateColumn("scan_count", gorm.Expr("scan_count + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNoRoutingTarget
	}
	return nil
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
