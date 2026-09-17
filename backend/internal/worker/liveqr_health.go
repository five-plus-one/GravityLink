package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/service"
)

// LiveQRHealthWorker 周期巡检活跃群活码：即将过期/已过期/接近阈值/已达阈值/整码无可用。
type LiveQRHealthWorker struct {
	db       *gorm.DB
	redis    *redis.Client
	notifier *service.Notifier
	logger   *slog.Logger
}

func NewLiveQRHealthWorker(db *gorm.DB, redis *redis.Client, notifier *service.Notifier, logger *slog.Logger) *LiveQRHealthWorker {
	return &LiveQRHealthWorker{db: db, redis: redis, notifier: notifier, logger: logger}
}

func (w *LiveQRHealthWorker) Start(ctx context.Context) {
	// 启动后稍等再跑，避免与初始化争抢
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}
	interval := w.interval(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil && w.logger != nil {
				w.logger.Warn("liveqr health check failed", "error", err)
			}
		}
	}
}

func (w *LiveQRHealthWorker) interval(ctx context.Context) time.Duration {
	cfg, err := w.notifier.LoadEvents(ctx)
	if err != nil || cfg.ExpiringLeadHours <= 0 {
		return 10 * time.Minute
	}
	// 巡检间隔不从 events 配置暴露独立字段时固定 10 分钟；
	// 未来可在 NotifyEventsConfig 增加字段。此处保持 10 分钟。
	_ = cfg
	return 10 * time.Minute
}

// RunOnce 供测试与手动触发。
func (w *LiveQRHealthWorker) RunOnce(ctx context.Context) error {
	if w.notifier == nil {
		return nil
	}
	eventsCfg, err := w.notifier.LoadEvents(ctx)
	if err != nil {
		return err
	}
	lead := time.Duration(eventsCfg.ExpiringLeadHours) * time.Hour
	if lead <= 0 {
		lead = 24 * time.Hour
	}
	ratio := eventsCfg.ScanNearRatio
	if ratio <= 0 || ratio >= 1 {
		ratio = 0.9
	}

	var links []model.Link
	if err := w.db.WithContext(ctx).
		Where("type = ? AND status = ?", model.LinkTypeLiveQR, model.StatusActive).
		Find(&links).Error; err != nil {
		return err
	}
	now := time.Now()
	for _, link := range links {
		if err := w.checkLink(ctx, link, now, lead, ratio); err != nil && w.logger != nil {
			w.logger.Warn("liveqr health link failed", "code", link.Code, "error", err)
		}
	}
	return nil
}

type healthTarget struct {
	ID        uint64
	Label     string
	ExpireAt  *time.Time
	ScanLimit *uint
	ScanCount uint
}

func (w *LiveQRHealthWorker) checkLink(ctx context.Context, link model.Link, now time.Time, lead time.Duration, ratio float64) error {
	var strategy model.RoutingStrategy
	err := w.db.WithContext(ctx).Where("link_id = ?", link.ID).First(&strategy).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	var targets []model.RoutingTarget
	if err := w.db.WithContext(ctx).
		Where("strategy_id = ? AND status = ?", strategy.ID, model.StatusActive).
		Find(&targets).Error; err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	linkTitle := ""
	if link.Title != nil {
		linkTitle = *link.Title
	}
	available := 0
	reasons := map[string]int{}

	for _, t := range targets {
		label := "未命名"
		if t.Label != nil && strings.TrimSpace(*t.Label) != "" {
			label = strings.TrimSpace(*t.Label)
		}
		scanLimit := "—"
		if t.ScanLimit != nil {
			scanLimit = fmt.Sprintf("%d", *t.ScanLimit)
		}
		vars := service.NotifyVars{
			"LinkCode":    link.Code,
			"LinkTitle":   linkTitle,
			"LinkURL":     "",
			"TargetLabel": label,
			"TargetID":    fmt.Sprintf("%d", t.ID),
			"ScanCount":   fmt.Sprintf("%d", t.ScanCount),
			"ScanLimit":   scanLimit,
			"Time":        now.Format("2006-01-02 15:04"),
		}

		expired := t.ExpireAt != nil && !t.ExpireAt.After(now)
		nearExpire := t.ExpireAt != nil && t.ExpireAt.After(now) && t.ExpireAt.Sub(now) <= lead
		atLimit := t.ScanLimit != nil && t.ScanCount >= *t.ScanLimit
		nearLimit := t.ScanLimit != nil && !atLimit && float64(t.ScanCount) >= float64(*t.ScanLimit)*ratio

		if expired {
			reasons["到期"]++
			vars["ExpireAt"] = t.ExpireAt.Format("2006-01-02 15:04")
			// 「启用且未达阈值但已过期」是重点
			if !atLimit {
				key := fmt.Sprintf("expired:%d", t.ID)
				w.notifier.NotifyAsync(service.EventTargetExpired, key, vars)
			}
		} else {
			available++
			if nearExpire {
				vars["ExpireAt"] = t.ExpireAt.Format("2006-01-02 15:04")
				vars["ExpireIn"] = service.FormatExpireIn(t.ExpireAt.Sub(now))
				key := fmt.Sprintf("expiring:%d:%s", t.ID, now.Format("20060102"))
				w.notifier.NotifyAsync(service.EventTargetExpiringSoon, key, vars)
			}
		}
		if atLimit {
			reasons["满额"]++
			key := fmt.Sprintf("scan_exhausted:%d", t.ID)
			w.notifier.NotifyAsync(service.EventTargetScanExhausted, key, vars)
		} else if nearLimit {
			key := fmt.Sprintf("scan_near:%d", t.ID)
			w.notifier.NotifyAsync(service.EventTargetScanNear, key, vars)
		}
	}

	// 可分发口径与 routing.filterAvailable 一致：未到期且未满额
	// 上面 available 在 else 分支累加（未到期）；但未到期也可能已满额，需修正
	available = 0
	reasons = map[string]int{}
	for _, t := range targets {
		expired := t.ExpireAt != nil && !t.ExpireAt.After(now)
		atLimit := t.ScanLimit != nil && t.ScanCount >= *t.ScanLimit
		if expired {
			reasons["到期"]++
		} else if atLimit {
			reasons["满额"]++
		} else {
			available++
		}
	}

	if available == 0 {
		var reasonParts []string
		for k, v := range reasons {
			reasonParts = append(reasonParts, fmt.Sprintf("%s %d", k, v))
		}
		if len(reasonParts) == 0 {
			reasonParts = append(reasonParts, "无可用目标")
		}
		vars := service.NotifyVars{
			"LinkCode":       link.Code,
			"LinkTitle":      linkTitle,
			"AvailableCount": "0",
			"TotalCount":     fmt.Sprintf("%d", len(targets)),
			"Reason":         strings.Join(reasonParts, "、"),
			"Time":           now.Format("2006-01-02 15:04"),
		}
		key := "qr_exhausted:" + link.Code
		w.notifier.NotifyAsync(service.EventLiveQRNoAvailable, key, vars)
	}
	return nil
}
