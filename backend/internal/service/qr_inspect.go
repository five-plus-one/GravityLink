package service

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// QR 识别结果
const (
	QRKindWeChatGroup  = "wechat_group"
	QRKindWeChatOther  = "wechat_other"
	QRKindURL          = "url"
	QRKindUnknown      = "unknown"
	QRKindNotQR        = "not_qr"
)

const (
	ExpireSourceURLParam         = "url_param"
	ExpireSourceWeChatGroupDef   = "wechat_group_default"
	ExpireSourceWeChatPayload    = "wechat_payload"
	ExpireSourceNone             = "none"
)

type QRInspectResult struct {
	Content           string     `json:"content"`
	Kind              string     `json:"kind"`
	SuggestedExpireAt *time.Time `json:"suggested_expire_at"`
	ExpireSource      string     `json:"expire_source"`
}

// InspectQR 解码二维码并估算过期时间。失败时返回 not_qr，不报错中断业务。
func InspectQR(data []byte, defaultDays int) QRInspectResult {
	if defaultDays <= 0 {
		defaultDays = 7
	}
	result := QRInspectResult{Kind: QRKindNotQR, ExpireSource: ExpireSourceNone}
	if len(data) == 0 || len(data) > 5<<20 {
		return result
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return result
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return result
	}
	text, err := decodeQRBitmap(bmp)
	if err != nil || strings.TrimSpace(text) == "" {
		return result
	}
	result.Content = truncateRunes(text, 2048)
	result.Kind, result.SuggestedExpireAt, result.ExpireSource = classifyQRContent(text, time.Now(), defaultDays)
	return result
}

func decodeQRBitmap(bmp *gozxing.BinaryBitmap) (string, error) {
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}
	result, err := qrcode.NewQRCodeReader().Decode(bmp, hints)
	if err != nil {
		return "", err
	}
	return result.GetText(), nil
}

func classifyQRContent(content string, now time.Time, defaultDays int) (kind string, expireAt *time.Time, source string) {
	text := strings.TrimSpace(content)
	lower := strings.ToLower(text)

	// URL 参数过期
	if u, err := url.Parse(text); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		q := u.Query()
		for _, key := range []string{"expire", "expires", "expiry", "exp", "expire_at", "expireat"} {
			raw := strings.TrimSpace(q.Get(key))
			if raw == "" {
				continue
			}
			if ts, ok := parseExpireValue(raw, now); ok {
				return classifyWeChatURL(lower), &ts, ExpireSourceURLParam
			}
		}
		// 无参数时按微信特征判断
		k := classifyWeChatURL(lower)
		if k == QRKindWeChatGroup {
			t := now.Add(time.Duration(defaultDays) * 24 * time.Hour)
			return k, &t, ExpireSourceWeChatGroupDef
		}
		return k, nil, ExpireSourceNone
	}

	// 非 URL：微信特征（含 weixin.qq.com/g/ 子串的编码体等）
	if isWeChatGroupPayload(lower) {
		t := now.Add(time.Duration(defaultDays) * 24 * time.Hour)
		return QRKindWeChatGroup, &t, ExpireSourceWeChatGroupDef
	}
	if isWeChatPayload(lower) {
		return QRKindWeChatOther, nil, ExpireSourceNone
	}
	// 启发式：正文中出现 10 位秒级时间戳且落在合理区间
	if ts, ok := extractEmbeddedUnix(text, now); ok {
		return QRKindWeChatOther, &ts, ExpireSourceWeChatPayload
	}
	return QRKindUnknown, nil, ExpireSourceNone
}

func classifyWeChatURL(lower string) string {
	if strings.Contains(lower, "weixin.qq.com/g/") || strings.Contains(lower, "wx.qq.com/g/") {
		return QRKindWeChatGroup
	}
	if strings.Contains(lower, "weixin.qq.com") || strings.Contains(lower, "wx.qq.com") || strings.Contains(lower, "qq.com/cgi-bin/qrconnect") {
		return QRKindWeChatOther
	}
	return QRKindURL
}

func isWeChatGroupPayload(lower string) bool {
	return strings.Contains(lower, "weixin.qq.com/g/") ||
		strings.Contains(lower, "wx.qq.com/g/") ||
		(strings.Contains(lower, "weixin") && strings.Contains(lower, "/g/"))
}

func isWeChatPayload(lower string) bool {
	return strings.Contains(lower, "weixin.qq.com") ||
		strings.Contains(lower, "wx.qq.com") ||
		strings.HasPrefix(lower, "wxp://") ||
		strings.Contains(lower, "qrconnect")
}

func parseExpireValue(raw string, now time.Time) (time.Time, bool) {
	// 纯数字：秒或毫秒
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if n > 1e12 { // 毫秒
			n = n / 1000
		}
		if n > 1_000_000_000 && n < 4_000_000_000 {
			t := time.Unix(n, 0)
			// 合理窗口：过去 30 天 ~ 未来 10 年
			if t.After(now.Add(-30*24*time.Hour)) && t.Before(now.AddDate(10, 0, 0)) {
				return t, true
			}
		}
		return time.Time{}, false
	}
	// RFC3339 / 常见格式
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func extractEmbeddedUnix(text string, now time.Time) (time.Time, bool) {
	// 扫描连续 10 位数字
	for i := 0; i+10 <= len(text); i++ {
		chunk := text[i : i+10]
		okDigits := true
		for _, c := range chunk {
			if c < '0' || c > '9' {
				okDigits = false
				break
			}
		}
		if !okDigits {
			continue
		}
		n, err := strconv.ParseInt(chunk, 10, 64)
		if err != nil {
			continue
		}
		if n > 1_600_000_000 && n < 2_000_000_000 {
			t := time.Unix(n, 0)
			// 判定为「生成时间」而非过期：若在最近 30 天内，则 +7 天视为过期
			if t.After(now.Add(-30*24*time.Hour)) && t.Before(now.Add(24*time.Hour)) {
				exp := t.Add(7 * 24 * time.Hour)
				return exp, true
			}
			// 直接是过期时间
			if t.After(now) && t.Before(now.AddDate(2, 0, 0)) {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
