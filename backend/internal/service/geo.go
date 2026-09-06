package service

import (
	"log/slog"
	"strings"

	ip2region "github.com/lionsoul2014/ip2region/binding/golang/service"
)

// GeoInfo IP 归属地信息，空字符串表示未知。
type GeoInfo struct {
	Country  string
	Province string
	City     string
	ISP      string
}

// GeoFieldSep stat:geo Hash field 与 stat_geo 唯一键的分隔符（country|province）。
const GeoFieldSep = "|"

// GeoResolver IP 归属地解析接口。
type GeoResolver interface {
	Resolve(ip string) GeoInfo
}

// NoopGeoResolver 空实现：未配置 IP 库或加载失败时使用，解析结果全空。
type NoopGeoResolver struct{}

func (NoopGeoResolver) Resolve(string) GeoInfo { return GeoInfo{} }

// ip2regionResolver 基于 ip2region xdb 的解析实现。
// 底层 Ip2Region 通过 searcher pool 支持并发查询。
type ip2regionResolver struct {
	searcher *ip2region.Ip2Region
}

// NewGeoResolver 按路径创建 IP 归属地解析器。
// 路径为空或文件加载失败时降级为 NoopGeoResolver（记日志，不阻断启动）。
func NewGeoResolver(path string, logger *slog.Logger) GeoResolver {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(path) == "" {
		logger.Info("geo resolver disabled: GEO_DB_PATH not configured")
		return NoopGeoResolver{}
	}
	searcher, err := ip2region.NewIp2RegionWithPath(path, "")
	if err != nil {
		logger.Warn("geo resolver disabled: failed to load ip db", "path", path, "error", err)
		return NoopGeoResolver{}
	}
	logger.Info("geo resolver enabled", "path", path)
	return &ip2regionResolver{searcher: searcher}
}

func (r *ip2regionResolver) Resolve(ip string) GeoInfo {
	if strings.TrimSpace(ip) == "" {
		return GeoInfo{}
	}
	region, err := r.searcher.Search(ip)
	if err != nil {
		return GeoInfo{}
	}
	return ParseGeoRegion(region)
}

// ParseGeoRegion 解析 ip2region v3 返回的 "国家|省份|城市|ISP|iso-alpha2-code" 格式，
// 缺失字段以 "0" 占位，统一归一为空字符串。
func ParseGeoRegion(region string) GeoInfo {
	parts := strings.Split(region, "|")
	value := func(i int) string {
		if i >= len(parts) || parts[i] == "0" {
			return ""
		}
		return parts[i]
	}
	return GeoInfo{
		Country:  value(0),
		Province: value(1),
		City:     value(2),
		ISP:      value(3),
	}
}
