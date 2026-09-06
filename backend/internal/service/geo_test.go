package service

import "testing"

func TestParseGeoRegion(t *testing.T) {
	// ip2region v3 格式："国家|省份|城市|ISP|iso-alpha2-code"，缺失以 "0" 占位
	tests := []struct {
		name     string
		region   string
		country  string
		province string
		city     string
		isp      string
	}{
		{"国内完整记录", "中国|江苏省|南京市|电信|CN", "中国", "江苏省", "南京市", "电信"},
		{"省份缺失城市直填", "中国|0|南京市||CN", "中国", "", "南京市", ""},
		{"ISP 为组织名", "中国|0|杭州市|阿里|CN", "中国", "", "杭州市", "阿里"},
		{"国外记录", "United States|0|0|Google LLC|US", "United States", "", "", "Google LLC"},
		{"全缺失", "0|0|0|0|0", "", "", "", ""},
		{"字段不足", "中国|0", "中国", "", "", ""},
		{"空字符串", "", "", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseGeoRegion(tt.region)
			if got.Country != tt.country || got.Province != tt.province || got.City != tt.city || got.ISP != tt.isp {
				t.Errorf("ParseGeoRegion(%q) = %+v, want country=%q province=%q city=%q isp=%q",
					tt.region, got, tt.country, tt.province, tt.city, tt.isp)
			}
		})
	}
}

func TestNoopGeoResolver(t *testing.T) {
	got := NoopGeoResolver{}.Resolve("1.2.3.4")
	if got != (GeoInfo{}) {
		t.Errorf("NoopGeoResolver should return empty GeoInfo, got %+v", got)
	}
}
