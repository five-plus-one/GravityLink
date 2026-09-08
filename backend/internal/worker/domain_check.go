package worker

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/service"
)

// DomainCheckWorker 定时检测活跃域名是否被微信拦截（借 mp.weixinbridge.com 官方桥接接口，
// 响应 Location 含 weixin110 即判定被封，与旧版引流宝同一原理）。
// 状态存 Redis（domain:wxban:{host}）；从「正常」变为「被封」时通过 Notifier 告警。
// 通过 system_configs 的 domain_check_enabled 开关控制，默认关闭。
type DomainCheckWorker struct {
	db       *gorm.DB
	redis    *redis.Client
	notifier *service.Notifier
	logger   *slog.Logger
	client   *http.Client
}

func NewDomainCheckWorker(db *gorm.DB, redis *redis.Client, notifier *service.Notifier, logger *slog.Logger) *DomainCheckWorker {
	return &DomainCheckWorker{
		db: db, redis: redis, notifier: notifier, logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
			// 不跟随跳转：靠读取 Location 判断 weixin110 特征
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (w *DomainCheckWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !w.enabled(ctx) {
				continue
			}
			if err := w.checkAll(ctx); err != nil {
				w.logger.Warn("domain check failed", "error", err)
			}
		}
	}
}

func (w *DomainCheckWorker) enabled(ctx context.Context) bool {
	configs, err := service.NewSystemConfigService(w.db).List(ctx)
	if err != nil {
		return false
	}
	return configs[service.DomainCheckEnabled] == "1" || strings.EqualFold(configs[service.DomainCheckEnabled], "true")
}

func (w *DomainCheckWorker) checkAll(ctx context.Context) error {
	var domains []model.Domain
	if err := w.db.WithContext(ctx).Where("status = ?", model.StatusActive).Find(&domains).Error; err != nil {
		return err
	}
	for _, domain := range domains {
		banned := w.checkHost(ctx, domain.Host)
		key := "domain:wxban:" + domain.Host
		prev, _ := w.redis.Get(ctx, key).Result()
		wasBanned := prev == "1"

		if banned {
			w.redis.Set(ctx, key, 1, 0)
			if !wasBanned && w.notifier != nil {
				w.notifier.SendAsync(
					"domain_ban:"+domain.Host,
					"⚠ 域名被微信封禁",
					"域名 "+domain.Host+"（"+domain.Type+"）被微信拦截，请尽快更换域名！",
				)
			}
		} else {
			w.redis.Set(ctx, key, 0, 0)
		}
		// 检测间隔 2 秒，避免触发对方接口风控
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil
}

func (w *DomainCheckWorker) checkHost(ctx context.Context, host string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://mp.weixinbridge.com/mp/wapredirect?url="+host, nil)
	if err != nil {
		return false
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return strings.Contains(resp.Header.Get("Location"), "weixin110")
}
