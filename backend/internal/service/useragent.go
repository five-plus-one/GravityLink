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

// CheckAccessRule 校验 UA 是否满足链接的访问限制。
// rule 来自 Link.AccessRule（none/wechat/ios/android/mobile/pc）。
// 返回允许（true，""）或拒绝（false，限制说明提示语）。
func CheckAccessRule(ua, rule string) (allowed bool, hint string) {
	uaLower := strings.ToLower(ua)
	switch rule {
	case "wechat":
		if strings.Contains(uaLower, "micromessenger") {
			return true, ""
		}
		return false, "请使用微信客户端打开此链接"
	case "ios":
		if containsAny(uaLower, []string{"iphone", "ipad", "ipod"}) {
			return true, ""
		}
		return false, "请使用 iOS 设备打开此链接"
	case "android":
		if strings.Contains(uaLower, "android") {
			return true, ""
		}
		return false, "请使用 Android 设备打开此链接"
	case "mobile":
		if parseDevice(uaLower) == "mobile" {
			return true, ""
		}
		return false, "请使用手机浏览器打开此链接"
	case "pc":
		if parseDevice(uaLower) == "desktop" {
			return true, ""
		}
		return false, "请使用电脑浏览器打开此链接"
	default:
		return true, ""
	}
}

// ParseSourceApp 通过 Referer 域名与 UA 特征识别访问来源 APP，供渠道码统计分析。
// 返回中文名称（微信/抖音/小红书/微博/B站/QQ/淘宝/百度/知乎/快手/Safari/Chrome 等），
// 无法识别时返回空字符串。
func ParseSourceApp(ua, referer string) string {
	uaLow := strings.ToLower(ua)
	refererLow := strings.ToLower(referer)

	// 先检查 UA 特征（覆盖 Referer 为空的场景：微信/QQ 内分享不带 Referer）
	if strings.Contains(uaLow, "micromessenger") {
		return "微信"
	}
	if strings.Contains(uaLow, "mqqbrowser") || strings.Contains(uaLow, " qq/") {
		return "QQ"
	}
	if strings.Contains(uaLow, "douyin") {
		return "抖音"
	}
	if strings.Contains(uaLow, "toutiao") {
		return "今日头条"
	}
	if strings.Contains(uaLow, "xiaohongshu") || strings.Contains(uaLow, "rednotes") {
		return "小红书"
	}
	if strings.Contains(uaLow, "kuaishou") {
		return "快手"
	}

	// 从 Referer 域名映射（覆盖外部浏览器通过第三方 APP 打开的场景）
	if refererLow != "" {
		host := refererDomain(refererLow)
		switch {
		case strings.Contains(host, "weibo.com") || strings.Contains(host, "weibo.cn"):
			return "微博"
		case strings.Contains(host, "bilibili.com"):
			return "B站"
		case strings.Contains(host, "xiaohongshu.com"):
			return "小红书"
		case strings.Contains(host, "taobao.com") || strings.Contains(host, "tmall.com"):
			return "淘宝"
		case strings.Contains(host, "alipay.com"):
			return "支付宝"
		case strings.Contains(host, "baidu.com"):
			return "百度"
		case strings.Contains(host, "zhihu.com"):
			return "知乎"
		case strings.Contains(host, "douyin.com") || strings.Contains(host, "tiktok.com"):
			return "抖音"
		case strings.Contains(host, "kuaishou.com"):
			return "快手"
		case strings.Contains(host, "weixin.qq.com") || strings.Contains(host, "mp.weixin.qq.com"):
			return "微信"
		case strings.Contains(host, "twitter.com") || strings.Contains(host, "x.com"):
			return "X/Twitter"
		case strings.Contains(host, "instagram.com"):
			return "Instagram"
		case strings.Contains(host, "facebook.com"):
			return "Facebook"
		case strings.Contains(host, "youtube.com"):
			return "YouTube"
		}
	}

	return ""
}

// refererDomain 从 URL 提取 host 部分。
func refererDomain(referer string) string {
	start := 0
	if strings.HasPrefix(referer, "http://") {
		start = 7
	} else if strings.HasPrefix(referer, "https://") {
		start = 8
	}
	end := strings.IndexAny(referer[start:], "/?")
	if end < 0 {
		return referer[start:]
	}
	return referer[start : start+end]
}
