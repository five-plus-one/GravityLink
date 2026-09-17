package service

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRenderNotifyTemplateDefaults(t *testing.T) {
	cfg := DefaultNotifyEvents()
	vars := NotifyVars{
		"LinkCode":    "abc123",
		"LinkTitle":   "测试活码",
		"TargetLabel": "群1",
		"ExpireAt":    "2026-09-17 10:00",
		"ExpireIn":    "23小时",
		"ScanCount":   "10",
		"ScanLimit":   "200",
	}
	title, body, err := RenderNotifyTemplate(cfg, EventTargetExpiringSoon, vars)
	if err != nil {
		t.Fatal(err)
	}
	if title != "群活码即将过期" {
		t.Fatalf("title=%q", title)
	}
	if !strings.Contains(body, "测试活码") || !strings.Contains(body, "群1") || !strings.Contains(body, "23小时") {
		t.Fatalf("body=%q", body)
	}
}

func TestRenderNotifyTemplateNoTitleFallsBackToCode(t *testing.T) {
	cfg := DefaultNotifyEvents()
	vars := NotifyVars{"LinkCode": "code9", "TargetLabel": "A", "ExpireAt": "x", "ExpireIn": "y", "ScanCount": "1", "ScanLimit": "—"}
	_, body, err := RenderNotifyTemplate(cfg, EventTargetExpired, vars)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "code9") {
		t.Fatalf("body=%q", body)
	}
}

func TestClassifyQRContentWeChatGroupDefault(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.Local)
	kind, exp, src := classifyQRContent("https://weixin.qq.com/g/AbCdEf123", now, 7)
	if kind != QRKindWeChatGroup {
		t.Fatalf("kind=%s", kind)
	}
	if src != ExpireSourceWeChatGroupDef {
		t.Fatalf("src=%s", src)
	}
	if exp == nil || exp.Sub(now) < 6*24*time.Hour || exp.Sub(now) > 8*24*time.Hour {
		t.Fatalf("expire=%v", exp)
	}
}

func TestClassifyQRContentURLParam(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.Local)
	ts := now.Add(48 * time.Hour).Unix()
	kind, exp, src := classifyQRContent("https://example.com/join?expire="+itoa(ts), now, 7)
	if kind != QRKindURL {
		t.Fatalf("kind=%s", kind)
	}
	if src != ExpireSourceURLParam || exp == nil {
		t.Fatalf("src=%s exp=%v", src, exp)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

func TestClassifyQRContentNotQR(t *testing.T) {
	res := InspectQR([]byte("not an image"), 7)
	if res.Kind != QRKindNotQR {
		t.Fatalf("kind=%s", res.Kind)
	}
}

func TestFormatExpireIn(t *testing.T) {
	if FormatExpireIn(2*time.Hour) != "2小时" {
		t.Fatal(FormatExpireIn(2 * time.Hour))
	}
	if FormatExpireIn(3*24*time.Hour) != "3天" {
		t.Fatal(FormatExpireIn(3 * 24 * time.Hour))
	}
}

func TestBuildSMTPMessage(t *testing.T) {
	msg := string(buildSMTPMessage("a@qq.com", []string{"b@qq.com"}, "测试标题", "你好"))
	if !strings.Contains(msg, "Subject: ") || !strings.Contains(msg, "Content-Transfer-Encoding: base64") {
		t.Fatalf("msg=%s", msg)
	}
	if !strings.Contains(msg, "To: b@qq.com") {
		t.Fatalf("missing to")
	}
}

func TestFormatFromHeader(t *testing.T) {
	h := formatFromHeader("GravityLink", "noreply@example.com")
	if !strings.Contains(h, "<noreply@example.com>") || !strings.Contains(h, "GravityLink") {
		t.Fatalf("header=%q", h)
	}
	// 纯邮箱
	if formatFromHeader("", "a@b.com") != "a@b.com" {
		t.Fatal("bare email")
	}
	// 已含尖括号则透传
	full := formatFromHeader("X", "Name <a@b.com>")
	if full != "Name <a@b.com>" {
		t.Fatalf("passthrough=%q", full)
	}
}

func TestParseFrom(t *testing.T) {
	name, email := parseFrom("GravityLink <noreply@example.com>", "")
	if name != "GravityLink" || email != "noreply@example.com" {
		t.Fatalf("name=%q email=%q", name, email)
	}
	name, email = parseFrom("noreply@example.com", "默认名")
	if name != "默认名" || email != "noreply@example.com" {
		t.Fatalf("name=%q email=%q", name, email)
	}
}

func TestDebounceTTL(t *testing.T) {
	if DebounceTTL(EventTargetExpiringSoon) != 24*time.Hour {
		t.Fatal("expiring ttl")
	}
	if DebounceTTL(EventLiveQRNoAvailable) != time.Hour {
		t.Fatal("no available ttl")
	}
}
