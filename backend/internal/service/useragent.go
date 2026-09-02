package service

import "strings"

// UserAgentInfo 是解析后的用户代理信息。
// Device 取值与 access_logs.device 枚举一致：mobile/tablet/desktop/bot/unknown。
type UserAgentInfo struct {
	Device  string
	OS      string
	Browser string
}

// bot 特征：常见爬虫与命令行 HTTP 客户端。
var botKeywords = []string{
	"bot", "spider", "crawl", "slurp", "headlesschrome",
	"curl", "wget", "python-requests", "python-urllib", "go-http-client",
	"java/", "okhttp", "httpclient", "axios", "node-fetch", "postmanruntime",
}

// ParseUserAgent 通过关键词解析 UA，返回设备类型、操作系统与浏览器。
// 识别失败的 OS/Browser 返回空字符串，Device 返回 unknown（UA 为空）或 desktop。
func ParseUserAgent(ua string) UserAgentInfo {
	ua = strings.ToLower(strings.TrimSpace(ua))
	if ua == "" {
		return UserAgentInfo{Device: "unknown"}
	}

	if containsAny(ua, botKeywords) {
		return UserAgentInfo{Device: "bot"}
	}

	return UserAgentInfo{
		Device:  parseDevice(ua),
		OS:      parseOS(ua),
		Browser: parseBrowser(ua),
	}
}

func parseDevice(ua string) string {
	switch {
	case containsAny(ua, []string{"ipad", "tablet"}):
		return "tablet"
	case strings.Contains(ua, "android") && !strings.Contains(ua, "mobile"):
		// Android 且不带 Mobile 标记的通常是平板
		return "tablet"
	case containsAny(ua, []string{"mobile", "android", "iphone", "ipod"}):
		return "mobile"
	default:
		return "desktop"
	}
}

func parseOS(ua string) string {
	switch {
	case containsAny(ua, []string{"iphone", "ipad", "ipod"}):
		return "iOS"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case containsAny(ua, []string{"mac os x", "macintosh"}):
		return "macOS"
	case containsAny(ua, []string{"linux", "x11"}):
		return "Linux"
	default:
		return ""
	}
}

func parseBrowser(ua string) string {
	switch {
	// 顺序敏感：微信/QQ 内置 UA 同时携带其他浏览器标记，需先判断
	case strings.Contains(ua, "micromessenger"):
		return "WeChat"
	case strings.Contains(ua, "mqqbrowser"):
		return "QQ"
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edge"):
		return "Edge"
	case strings.Contains(ua, "firefox") || strings.Contains(ua, "fxios"):
		return "Firefox"
	case strings.Contains(ua, "chrome") || strings.Contains(ua, "crios"):
		return "Chrome"
	case strings.Contains(ua, "safari"):
		return "Safari"
	default:
		return ""
	}
}

func containsAny(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}
