package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	AccessStreamKey = "access:stream"

	// DeviceStatKeyPrefix 设备维度计数 Hash key 前缀，field 为 "device|os|browser"。
	DeviceStatKeyPrefix = "stat:dev:"

	// GeoStatKeyPrefix 地域维度计数 Hash key 前缀，field 为 "country|province"。
	GeoStatKeyPrefix = "stat:geo:"
)

// DeviceStatKey 返回某链接某天的设备维度计数 Hash key。
func DeviceStatKey(linkID uint64, day string) string {
	return fmt.Sprintf("%s%d:%s", DeviceStatKeyPrefix, linkID, day)
}

// GeoStatKey 返回某链接某天的地域维度计数 Hash key。
func GeoStatKey(linkID uint64, day string) string {
	return fmt.Sprintf("%s%d:%s", GeoStatKeyPrefix, linkID, day)
}

type AccessRecorder struct {
	redis *redis.Client
	geo   GeoResolver
}

type AccessEvent struct {
	LinkID     uint64    `json:"link_id"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"ua"`
	Referer    string    `json:"referer"`
	VisitedAt  time.Time `json:"visited_at"`
	ViaTransit bool      `json:"via_transit"`
	Country    string    `json:"country,omitempty"`
	Province   string    `json:"province,omitempty"`
	City       string    `json:"city,omitempty"`
	ISP        string    `json:"isp,omitempty"`
}

func NewAccessRecorder(redis *redis.Client, geo GeoResolver) *AccessRecorder {
	if geo == nil {
		geo = NoopGeoResolver{}
	}
	return &AccessRecorder{redis: redis, geo: geo}
}

func (r *AccessRecorder) RecordAsync(event AccessEvent) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		geoInfo := r.geo.Resolve(event.IP)
		event.Country = geoInfo.Country
		event.Province = geoInfo.Province
		event.City = geoInfo.City
		event.ISP = geoInfo.ISP

		day := event.VisitedAt.Format("20060102")
		hour := event.VisitedAt.Format("15")
		_ = r.redis.Incr(ctx, fmt.Sprintf("stat:pv:%d:%s", event.LinkID, day)).Err()
		_ = r.redis.PFAdd(ctx, fmt.Sprintf("stat:uv:%d:%s", event.LinkID, day), event.IP).Err()
		_ = r.redis.Incr(ctx, fmt.Sprintf("stat:hourly:%d:%s:%s", event.LinkID, day, hour)).Err()

		info := ParseUserAgent(event.UserAgent)
		_ = r.redis.HIncrBy(ctx, DeviceStatKey(event.LinkID, day), info.Device+"|"+info.OS+"|"+info.Browser, 1).Err()

		if geoInfo.Country != "" || geoInfo.Province != "" {
			_ = r.redis.HIncrBy(ctx, GeoStatKey(event.LinkID, day), geoInfo.Country+GeoFieldSep+geoInfo.Province, 1).Err()
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return
		}
		_ = r.redis.LPush(ctx, AccessStreamKey, payload).Err()
	}()
}
