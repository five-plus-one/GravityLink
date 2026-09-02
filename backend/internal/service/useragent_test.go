package service

import "testing"

func TestParseUserAgent(t *testing.T) {
	tests := []struct {
		name    string
		ua      string
		device  string
		os      string
		browser string
	}{
		{"空 UA", "", "unknown", "", ""},
		{"Windows Chrome", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "desktop", "Windows", "Chrome"},
		{"Windows Edge", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0", "desktop", "Windows", "Edge"},
		{"Windows Firefox", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0", "desktop", "Windows", "Firefox"},
		{"macOS Safari", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15", "desktop", "macOS", "Safari"},
		{"Android Chrome", "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36", "mobile", "Android", "Chrome"},
		{"Android 平板无 Mobile 标记", "Mozilla/5.0 (Linux; Android 12; SM-X200) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36", "tablet", "Android", "Chrome"},
		{"iPhone Safari", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1", "mobile", "iOS", "Safari"},
		{"iPad Safari", "Mozilla/5.0 (iPad; CPU OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1", "tablet", "iOS", "Safari"},
		{"微信内置 Android", "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Mobile Safari/537.36 MicroMessenger/8.0.42", "mobile", "Android", "WeChat"},
		{"微信内置 iOS", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.38", "mobile", "iOS", "WeChat"},
		{"QQ 内置浏览器", "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Mobile Safari/537.36 MQQBrowser/6.2", "mobile", "Android", "QQ"},
		{"iOS Chrome CriOS", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.0.0 Mobile/15E148 Safari/604.1", "mobile", "iOS", "Chrome"},
		{"Googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "bot", "", ""},
		{"curl", "curl/8.4.0", "bot", "", ""},
		{"Baiduspider", "Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)", "bot", "", ""},
		{"未识别桌面 UA", "SomeRandomFetcher/1.0", "desktop", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseUserAgent(tt.ua)
			if got.Device != tt.device {
				t.Errorf("Device = %q, want %q", got.Device, tt.device)
			}
			if got.OS != tt.os {
				t.Errorf("OS = %q, want %q", got.OS, tt.os)
			}
			if got.Browser != tt.browser {
				t.Errorf("Browser = %q, want %q", got.Browser, tt.browser)
			}
		})
	}
}
