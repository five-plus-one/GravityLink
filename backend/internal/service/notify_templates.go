package service

import (
	"fmt"
	"strings"
	"text/template"
	"time"
)

// 事件名常量
const (
	EventTargetExpiringSoon  = "target_expiring_soon"
	EventTargetExpired       = "target_expired"
	EventTargetScanNear      = "target_scan_near"
	EventTargetScanExhausted = "target_scan_exhausted"
	EventLiveQRNoAvailable   = "liveqr_no_available"
)

func DefaultNotifyEvents() NotifyEventsConfig {
	return NotifyEventsConfig{
		ExpiringLeadHours:   24,
		ScanNearRatio:       0.9,
		QRDefaultExpireDays: 7,
		Events: map[string]NotifyEventConfig{
			EventTargetExpiringSoon: {
				Enabled: true,
				Title:   "群活码即将过期",
				Body: `活码「{{.LinkTitle}}{{if not .LinkTitle}}{{.LinkCode}}{{end}}」
二维码「{{.TargetLabel}}」将于 {{.ExpireAt}} 过期（约 {{.ExpireIn}}）。
当前扫码 {{.ScanCount}}/{{.ScanLimit}}，请及时更换群二维码。`,
			},
			EventTargetExpired: {
				Enabled: true,
				Title:   "群活码二维码已过期",
				Body: `活码「{{.LinkCode}}」二维码「{{.TargetLabel}}」已于 {{.ExpireAt}} 过期。
扫码 {{.ScanCount}}/{{.ScanLimit}}（未达阈值但码已失效），请更换后启用。`,
			},
			EventTargetScanNear: {
				Enabled: true,
				Title:   "群活码扫码接近阈值",
				Body: `活码「{{.LinkCode}}」二维码「{{.TargetLabel}}」
扫码 {{.ScanCount}}/{{.ScanLimit}}，接近上限，请准备下一码。`,
			},
			EventTargetScanExhausted: {
				Enabled: true,
				Title:   "群活码二维码已达阈值",
				Body:    `活码「{{.LinkCode}}」二维码「{{.TargetLabel}}」已达阈值 {{.ScanLimit}}，将不再参与分发。`,
			},
			EventLiveQRNoAvailable: {
				Enabled: true,
				Title:   "活码暂无可用二维码",
				Body: `活码「{{.LinkCode}}」当前无可用二维码（可分发 {{.AvailableCount}}/{{.TotalCount}}）。
原因：{{.Reason}}。访客会看到「暂无可用群」，请尽快补充。`,
			},
		},
	}
}

func DebounceTTL(event string) time.Duration {
	switch event {
	case EventTargetExpiringSoon:
		return 24 * time.Hour
	case EventTargetExpired, EventTargetScanExhausted:
		return 7 * 24 * time.Hour
	case EventTargetScanNear:
		return 6 * time.Hour
	default:
		return time.Hour
	}
}

func RenderNotifyTemplate(cfg NotifyEventsConfig, event string, vars NotifyVars) (string, string, error) {
	ec, ok := cfg.Events[event]
	if !ok {
		def := DefaultNotifyEvents()
		ec = def.Events[event]
	}
	if vars == nil {
		vars = NotifyVars{}
	}
	// 空值兜底，避免模板输出怪异
	if vars["ScanLimit"] == "" {
		vars["ScanLimit"] = "—"
	}
	if vars["Time"] == "" {
		vars["Time"] = time.Now().Format("2006-01-02 15:04")
	}
	title := strings.TrimSpace(ec.Title)
	if title == "" {
		title = event
	}
	body := ec.Body
	if strings.TrimSpace(body) == "" {
		body = event
	}
	renderedTitle, err := renderText(title, vars)
	if err != nil {
		return title, body, fmt.Errorf("title template: %w", err)
	}
	renderedBody, err := renderText(body, vars)
	if err != nil {
		return title, body, fmt.Errorf("body template: %w", err)
	}
	return renderedTitle, renderedBody, nil
}

func renderText(text string, vars NotifyVars) (string, error) {
	t, err := template.New("notify").Option("missingkey=zero").Parse(text)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, map[string]string(vars)); err != nil {
		return "", err
	}
	return b.String(), nil
}

// FormatExpireIn 剩余时长的中文描述。
func FormatExpireIn(d time.Duration) string {
	if d < 0 {
		return "已过期"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分钟", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d小时", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 30 {
		return fmt.Sprintf("%d天", days)
	}
	return fmt.Sprintf("%d天", days)
}
