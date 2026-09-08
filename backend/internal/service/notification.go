package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 通知配置的 system_configs 键名。
const (
	NotifyWebhookURLKey = "notify_webhook_url" // 企业微信机器人 webhook 地址
	NotifyHTTPURLKey    = "notify_http_url"    // 自定义通知接口（POST JSON）
	DomainCheckEnabled  = "domain_check_enabled"
)

// Notifier 基于 system_configs 的通知发送器。
// 支持两种渠道：企业微信机器人 webhook、自定义 HTTP POST 接口。
// 所有发送均为异步且带事件级防抖（同一事件 1 小时至多一次），绝不阻塞业务路径。
type Notifier struct {
	db    *gorm.DB
	redis *redis.Client
	http  *http.Client
}

func NewNotifier(db *gorm.DB, redis *redis.Client) *Notifier {
	return &Notifier{
		db:    db,
		redis: redis,
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

// SendAsync 异步发送通知；eventKey 用于防抖（如 "qr_exhausted:{linkID}"、"domain_ban:{host}"）。
func (n *Notifier) SendAsync(eventKey, title, content string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := n.Send(ctx, eventKey, title, content); err != nil {
			fmt.Println("notify failed:", err)
		}
	}()
}

// Send 同步发送（带防抖）。未配置任何渠道时静默跳过。
func (n *Notifier) Send(ctx context.Context, eventKey, title, content string) error {
	configs, err := NewSystemConfigService(n.db).List(ctx)
	if err != nil {
		return err
	}
	webhook := configs[NotifyWebhookURLKey]
	httpURL := configs[NotifyHTTPURLKey]
	if webhook == "" && httpURL == "" {
		return nil
	}
	// 防抖：同一事件 1 小时只发一次（Redis 不可用时降级为直接发送）
	if n.redis != nil && eventKey != "" {
		ok, err := n.redis.SetNX(ctx, "notify:debounce:"+eventKey, 1, time.Hour).Result()
		if err == nil && !ok {
			return nil
		}
	}
	if webhook != "" {
		if err := n.postJSON(ctx, webhook, map[string]any{
			"msgtype": "text",
			"text":    map[string]string{"content": title + "\n" + content},
		}); err != nil {
			return fmt.Errorf("webhook: %w", err)
		}
	}
	if httpURL != "" {
		if err := n.postJSON(ctx, httpURL, map[string]any{
			"title": title, "content": content, "time": time.Now().Format(time.RFC3339),
		}); err != nil {
			return fmt.Errorf("http: %w", err)
		}
	}
	return nil
}

func (n *Notifier) postJSON(ctx context.Context, url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
