package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

// 通知相关 system_configs 键。
const (
	NotifyChannelsKey   = "notify.channels"
	NotifyEventsKey     = "notify.events"
	NotifyWebhookURLKey = "notify_webhook_url" // 旧键，只读兼容
	NotifyHTTPURLKey    = "notify_http_url"    // 旧键，只读兼容
)

const (
	NotifyTypeWecom = "wecom"
	NotifyTypeHTTP  = "http"
	NotifyTypeSMTP  = "smtp"
)

var (
	ErrNotify       = errors.New("通知配置无效")
	ErrNotifySecret = errors.New("通知密钥无法解密，请重新填写 SMTP 授权码后保存")
)

// Notifier 多渠道通知发送器（企微 Webhook / 自定义 HTTP / SMTP）。
type Notifier struct {
	db      *gorm.DB
	redis   *redis.Client
	dataDir string
	http    *http.Client
}

func NewNotifier(db *gorm.DB, redisClient *redis.Client, dataDir string) *Notifier {
	return &Notifier{
		db:      db,
		redis:   redisClient,
		dataDir: dataDir,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *Notifier) sealKey() ([]byte, error) {
	if n.dataDir == "" {
		return nil, ErrNotifySecret
	}
	return NewWechatService(n.db, n.dataDir).key()
}

func (n *Notifier) seal(value string) (string, error) {
	key, err := n.sealKey()
	if err != nil {
		return "", err
	}
	return sealSecret(key, value)
}

func (n *Notifier) open(sealed string) (string, error) {
	key, err := n.sealKey()
	if err != nil {
		return "", err
	}
	plain, err := openSecret(key, sealed)
	if err != nil {
		return "", ErrNotifySecret
	}
	return plain, nil
}

type NotifyChannel struct {
	Type     string         `json:"type"`
	Enabled  bool           `json:"enabled"`
	Settings map[string]any `json:"settings"`
}

type NotifyChannelView struct {
	Type             string         `json:"type"`
	Enabled          bool           `json:"enabled"`
	Settings         map[string]any `json:"settings"`
	SecretConfigured bool           `json:"secret_configured"`
}

type SaveChannelInput struct {
	Type     string         `json:"type"`
	Enabled  bool           `json:"enabled"`
	Settings map[string]any `json:"settings"`
}

type SaveChannelsInput struct {
	Channels []SaveChannelInput `json:"channels"`
}

func defaultSMTPSettingsMap() map[string]any {
	return map[string]any{
		"host":         "smtp.qq.com",
		"port":         465,
		"encryption":   "ssl",
		"from":         "",
		"from_name":    "GravityLink",
		"username":     "",
		"password_enc": "",
		"to":           []any{},
	}
}

// ListChannels 读取渠道（脱敏）；不存在时从旧扁平键迁移。
func (n *Notifier) ListChannels(ctx context.Context) ([]NotifyChannelView, error) {
	channels, err := n.loadChannels(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]NotifyChannelView, 0, len(channels))
	for _, ch := range channels {
		settings := map[string]any{}
		for k, v := range ch.Settings {
			if k == "password_enc" {
				continue
			}
			settings[k] = v
		}
		secretConfigured := true
		if ch.Type == NotifyTypeSMTP {
			s, _ := ch.Settings["password_enc"].(string)
			secretConfigured = s != ""
			settings["host"] = ch.Settings["host"]
			settings["port"] = ch.Settings["port"]
			settings["encryption"] = ch.Settings["encryption"]
			settings["from"] = ch.Settings["from"]
			if v, ok := ch.Settings["from_name"]; ok {
				settings["from_name"] = v
			} else {
				settings["from_name"] = "GravityLink"
			}
			settings["username"] = ch.Settings["username"]
			settings["to"] = ch.Settings["to"]
		}
		views = append(views, NotifyChannelView{
			Type:             ch.Type,
			Enabled:          ch.Enabled,
			Settings:         settings,
			SecretConfigured: secretConfigured,
		})
	}
	return views, nil
}

func (n *Notifier) loadChannels(ctx context.Context) ([]NotifyChannel, error) {
	var row model.SystemConfig
	err := n.db.WithContext(ctx).Where("key_name = ?", NotifyChannelsKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return n.migrateLegacyChannels(ctx)
	}
	if err != nil {
		return nil, err
	}
	var channels []NotifyChannel
	if err := json.Unmarshal([]byte(row.Value), &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (n *Notifier) migrateLegacyChannels(ctx context.Context) ([]NotifyChannel, error) {
	configs, err := NewSystemConfigService(n.db).List(ctx)
	if err != nil {
		return nil, err
	}
	channels := []NotifyChannel{
		{Type: NotifyTypeWecom, Enabled: false, Settings: map[string]any{"webhook_url": ""}},
		{Type: NotifyTypeHTTP, Enabled: false, Settings: map[string]any{"url": ""}},
		{Type: NotifyTypeSMTP, Enabled: false, Settings: defaultSMTPSettingsMap()},
	}
	if u := strings.TrimSpace(configs[NotifyWebhookURLKey]); u != "" {
		channels[0].Enabled = true
		channels[0].Settings["webhook_url"] = u
	}
	if u := strings.TrimSpace(configs[NotifyHTTPURLKey]); u != "" {
		channels[1].Enabled = true
		channels[1].Settings["url"] = u
	}
	if err := n.saveChannels(ctx, channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (n *Notifier) saveChannels(ctx context.Context, channels []NotifyChannel) error {
	b, err := json.Marshal(channels)
	if err != nil {
		return err
	}
	return n.db.WithContext(ctx).Exec(
		"INSERT INTO system_configs (key_name, value, updated_at) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = NOW()",
		NotifyChannelsKey, string(b),
	).Error
}

// SaveChannels 保存渠道；SMTP 授权码加密，空密码保留旧密文。
func (n *Notifier) SaveChannels(ctx context.Context, input SaveChannelsInput) ([]NotifyChannelView, error) {
	if len(input.Channels) == 0 {
		return nil, fmt.Errorf("%w: 至少保留一个渠道定义", ErrNotify)
	}
	existing, err := n.loadChannels(ctx)
	if err != nil {
		return nil, err
	}
	byType := map[string]NotifyChannel{}
	for _, ch := range existing {
		byType[ch.Type] = ch
	}
	out := make([]NotifyChannel, 0, len(input.Channels))
	for _, item := range input.Channels {
		switch item.Type {
		case NotifyTypeWecom, NotifyTypeHTTP, NotifyTypeSMTP:
		default:
			return nil, fmt.Errorf("%w: 不支持的渠道类型 %s", ErrNotify, item.Type)
		}
		ch := NotifyChannel{Type: item.Type, Enabled: item.Enabled, Settings: map[string]any{}}
		for k, v := range item.Settings {
			ch.Settings[k] = v
		}
		if item.Type == NotifyTypeSMTP {
			if err := n.normalizeSMTPSettings(&ch, byType[NotifyTypeSMTP]); err != nil {
				return nil, err
			}
		} else {
			urlKey := "webhook_url"
			if item.Type == NotifyTypeHTTP {
				urlKey = "url"
			}
			u, _ := ch.Settings[urlKey].(string)
			ch.Settings[urlKey] = strings.TrimSpace(u)
			if ch.Enabled && ch.Settings[urlKey] == "" {
				return nil, fmt.Errorf("%w: 启用该渠道时必须填写地址", ErrNotify)
			}
		}
		out = append(out, ch)
	}
	if err := n.saveChannels(ctx, out); err != nil {
		return nil, err
	}
	return n.ListChannels(ctx)
}

func (n *Notifier) normalizeSMTPSettings(ch *NotifyChannel, prev NotifyChannel) error {
	getStr := func(m map[string]any, key string) string {
		s, _ := m[key].(string)
		return strings.TrimSpace(s)
	}
	getInt := func(m map[string]any, key string, def int) int {
		switch v := m[key].(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			var num int
			if _, err := fmt.Sscanf(v, "%d", &num); err == nil {
				return num
			}
		}
		return def
	}
	host := getStr(ch.Settings, "host")
	if host == "" {
		host = "smtp.qq.com"
	}
	port := getInt(ch.Settings, "port", 465)
	if port != 465 && port != 587 && port != 25 {
		return fmt.Errorf("%w: SMTP 端口仅支持 465/587/25", ErrNotify)
	}
	enc := getStr(ch.Settings, "encryption")
	if enc == "" {
		if port == 465 {
			enc = "ssl"
		} else if port == 587 {
			enc = "starttls"
		} else {
			enc = "none"
		}
	}
	fromRaw := getStr(ch.Settings, "from")
	fromName := getStr(ch.Settings, "from_name")
	fromName, fromEmail := parseFrom(fromRaw, fromName)
	username := getStr(ch.Settings, "username")
	if username == "" {
		username = fromEmail
	}
	if fromEmail == "" {
		fromEmail = username
	}
	if fromName == "" {
		fromName = "GravityLink"
	}
	to := parseRecipients(ch.Settings["to"])
	if len(to) > 20 {
		return fmt.Errorf("%w: 收件人最多 20 个", ErrNotify)
	}

	password, _ := ch.Settings["password"].(string)
	password = strings.TrimSpace(password)
	var encPass string
	if password != "" {
		sealed, err := n.seal(password)
		if err != nil {
			return err
		}
		encPass = sealed
	} else if prev.Settings != nil {
		if s, ok := prev.Settings["password_enc"].(string); ok {
			encPass = s
		}
	}
	if ch.Enabled {
		if username == "" || encPass == "" || len(to) == 0 {
			return fmt.Errorf("%w: 邮件渠道启用时需填写账号、SMTP 授权码和至少一个收件人", ErrNotify)
		}
	}

	ch.Settings = map[string]any{
		"host":         host,
		"port":         port,
		"encryption":   enc,
		"from":         fromEmail,
		"from_name":    fromName,
		"username":     username,
		"password_enc": encPass,
		"to":           to,
	}
	return nil
}

func parseRecipients(raw any) []string {
	to := []string{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" {
			to = append(to, s)
		}
	}
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	case []string:
		for _, s := range v {
			add(s)
		}
	case string:
		for _, part := range strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\n'
		}) {
			add(part)
		}
	}
	return to
}

type NotifyEventConfig struct {
	Enabled bool   `json:"enabled"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

type NotifyEventsConfig struct {
	ExpiringLeadHours   int                          `json:"expiring_lead_hours"`
	ScanNearRatio       float64                      `json:"scan_near_ratio"`
	QRDefaultExpireDays int                          `json:"qr_default_expire_days"`
	Events              map[string]NotifyEventConfig `json:"events"`
}

type NotifyVars map[string]string

func (n *Notifier) LoadEvents(ctx context.Context) (NotifyEventsConfig, error) {
	cfg := DefaultNotifyEvents()
	var row model.SystemConfig
	err := n.db.WithContext(ctx).Where("key_name = ?", NotifyEventsKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal([]byte(row.Value), &cfg); err != nil {
		return cfg, err
	}
	defaults := DefaultNotifyEvents()
	if cfg.Events == nil {
		cfg.Events = defaults.Events
	} else {
		for k, v := range defaults.Events {
			if _, ok := cfg.Events[k]; !ok {
				cfg.Events[k] = v
			}
		}
	}
	if cfg.ExpiringLeadHours <= 0 {
		cfg.ExpiringLeadHours = defaults.ExpiringLeadHours
	}
	if cfg.ScanNearRatio <= 0 || cfg.ScanNearRatio >= 1 {
		cfg.ScanNearRatio = defaults.ScanNearRatio
	}
	if cfg.QRDefaultExpireDays <= 0 {
		cfg.QRDefaultExpireDays = defaults.QRDefaultExpireDays
	}
	return cfg, nil
}

func (n *Notifier) SaveEvents(ctx context.Context, cfg NotifyEventsConfig) (NotifyEventsConfig, error) {
	defaults := DefaultNotifyEvents()
	if cfg.ExpiringLeadHours <= 0 {
		cfg.ExpiringLeadHours = defaults.ExpiringLeadHours
	}
	if cfg.ScanNearRatio <= 0 || cfg.ScanNearRatio >= 1 {
		cfg.ScanNearRatio = defaults.ScanNearRatio
	}
	if cfg.QRDefaultExpireDays <= 0 {
		cfg.QRDefaultExpireDays = defaults.QRDefaultExpireDays
	}
	if cfg.Events == nil {
		cfg.Events = defaults.Events
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return cfg, err
	}
	if err := n.db.WithContext(ctx).Exec(
		"INSERT INTO system_configs (key_name, value, updated_at) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = NOW()",
		NotifyEventsKey, string(b),
	).Error; err != nil {
		return cfg, err
	}
	return cfg, nil
}

// TestChannel 向指定渠道发送测试通知（绕过事件防抖与订阅开关）。
func (n *Notifier) TestChannel(ctx context.Context, channelType string) error {
	channels, err := n.loadChannels(ctx)
	if err != nil {
		return err
	}
	var target *NotifyChannel
	for i := range channels {
		if channels[i].Type == channelType {
			target = &channels[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("%w: 渠道不存在", ErrNotify)
	}
	if !target.Enabled {
		// 测试允许对已配置但未启用渠道发送（便于先测通再打开）
		if target.Type == NotifyTypeSMTP {
			cfg, err := n.smtpRuntime(*target)
			if err != nil {
				return err
			}
			if cfg.Username == "" || cfg.Password == "" || len(cfg.To) == 0 {
				return fmt.Errorf("%w: 请先保存邮件账号、授权码和收件人", ErrNotify)
			}
		} else {
			u := ""
			if target.Type == NotifyTypeWecom {
				u, _ = target.Settings["webhook_url"].(string)
			} else {
				u, _ = target.Settings["url"].(string)
			}
			if strings.TrimSpace(u) == "" {
				return fmt.Errorf("%w: 请先保存该渠道地址", ErrNotify)
			}
		}
	}
	return n.sendOne(ctx, *target, "GravityLink 通知测试",
		"这是一条测试通知，发送时间："+time.Now().Format("2006-01-02 15:04:05")+"\n渠道："+channelType)
}

// Notify 按事件发送到所有已启用渠道（含模板与订阅开关）。
func (n *Notifier) Notify(ctx context.Context, event, eventKey string, vars NotifyVars) error {
	eventsCfg, err := n.LoadEvents(ctx)
	if err != nil {
		return err
	}
	if ec, ok := eventsCfg.Events[event]; ok && !ec.Enabled {
		return nil
	}
	title, body, err := RenderNotifyTemplate(eventsCfg, event, vars)
	if err != nil {
		return err
	}
	return n.Send(ctx, eventKey, title, body, DebounceTTL(event))
}

func (n *Notifier) NotifyAsync(event, eventKey string, vars NotifyVars) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := n.Notify(ctx, event, eventKey, vars); err != nil {
			fmt.Println("notify failed:", err)
		}
	}()
}

// Send 同步发送（带防抖）。
func (n *Notifier) Send(ctx context.Context, eventKey, title, content string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Hour
	}
	if n.redis != nil && eventKey != "" {
		ok, err := n.redis.SetNX(ctx, "notify:debounce:"+eventKey, 1, ttl).Result()
		if err == nil && !ok {
			return nil
		}
	}
	channels, err := n.loadChannels(ctx)
	if err != nil {
		return err
	}
	var lastErr error
	sent := 0
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		if err := n.sendOne(ctx, ch, title, content); err != nil {
			lastErr = err
			fmt.Println("notify channel failed:", ch.Type, err)
			continue
		}
		sent++
	}
	if sent == 0 && lastErr != nil {
		return lastErr
	}
	return nil
}

// SendAsync 兼容旧调用（固定 1h 防抖）。
func (n *Notifier) SendAsync(eventKey, title, content string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := n.Send(ctx, eventKey, title, content, time.Hour); err != nil {
			fmt.Println("notify failed:", err)
		}
	}()
}

func (n *Notifier) sendOne(ctx context.Context, ch NotifyChannel, title, content string) error {
	switch ch.Type {
	case NotifyTypeWecom:
		u, _ := ch.Settings["webhook_url"].(string)
		if strings.TrimSpace(u) == "" {
			return fmt.Errorf("webhook 地址为空")
		}
		return n.postJSON(ctx, u, map[string]any{
			"msgtype": "text",
			"text":    map[string]string{"content": title + "\n" + content},
		})
	case NotifyTypeHTTP:
		u, _ := ch.Settings["url"].(string)
		if strings.TrimSpace(u) == "" {
			return fmt.Errorf("HTTP 地址为空")
		}
		return n.postJSON(ctx, u, map[string]any{
			"title": title, "content": content, "time": time.Now().Format(time.RFC3339),
		})
	case NotifyTypeSMTP:
		cfg, err := n.smtpRuntime(ch)
		if err != nil {
			return err
		}
		return sendSMTPEmail(ctx, cfg, title, content)
	default:
		return fmt.Errorf("unsupported channel %s", ch.Type)
	}
}

func (n *Notifier) smtpRuntime(ch NotifyChannel) (SMTPSettings, error) {
	getStr := func(key string) string {
		s, _ := ch.Settings[key].(string)
		return strings.TrimSpace(s)
	}
	port := 465
	switch v := ch.Settings["port"].(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	}
	to := parseRecipients(ch.Settings["to"])
	enc, _ := ch.Settings["password_enc"].(string)
	password := ""
	if enc != "" {
		plain, err := n.open(enc)
		if err != nil {
			return SMTPSettings{}, err
		}
		password = plain
	}
	from := getStr("from")
	fromName := getStr("from_name")
	username := getStr("username")
	if username == "" {
		username = from
	}
	if from == "" {
		from = username
	}
	encType := getStr("encryption")
	if encType == "" {
		if port == 465 {
			encType = "ssl"
		} else if port == 587 {
			encType = "starttls"
		} else {
			encType = "none"
		}
	}
	return SMTPSettings{
		Host:       getStr("host"),
		Port:       port,
		Encryption: encType,
		From:       from,
		FromName:   fromName,
		Username:   username,
		Password:   password,
		To:         to,
	}, nil
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

// HasEnabledChannel 是否已配置可用通知渠道（用于上传后引导）。
func (n *Notifier) HasEnabledChannel(ctx context.Context) bool {
	channels, err := n.loadChannels(ctx)
	if err != nil {
		return false
	}
	for _, ch := range channels {
		if ch.Enabled {
			return true
		}
	}
	return false
}
