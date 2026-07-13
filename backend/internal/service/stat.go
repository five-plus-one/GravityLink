package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type StatService struct {
	db    *gorm.DB
	redis *redis.Client
}

type SummaryStats struct {
	TotalPV     uint64 `json:"total_pv"`
	TotalUV     uint64 `json:"total_uv"`
	TodayPV     uint64 `json:"today_pv"`
	TodayUV     uint64 `json:"today_uv"`
	YesterdayPV uint64 `json:"yesterday_pv"`
}

type DailyPoint struct {
	Date string `json:"date"`
	PV   uint64 `json:"pv"`
	UV   uint64 `json:"uv"`
}

type HourlyPoint struct {
	Hour uint8  `json:"hour"`
	PV   uint64 `json:"pv"`
}

type LabelValue struct {
	Label string `json:"label"`
	Value uint64 `json:"value"`
}

type DeviceStats struct {
	Device  []LabelValue `json:"device"`
	OS      []LabelValue `json:"os"`
	Browser []LabelValue `json:"browser"`
}

func NewStatService(db *gorm.DB, redis *redis.Client) *StatService {
	return &StatService{db: db, redis: redis}
}

func (s *StatService) Summary(ctx context.Context, linkID uint64) (SummaryStats, error) {
	var result SummaryStats
	type totals struct {
		PV uint64
		UV uint64
	}
	var total totals
	if err := s.db.WithContext(ctx).Table("stat_daily").Select("COALESCE(SUM(pv),0) AS pv, COALESCE(SUM(uv),0) AS uv").Where("link_id = ?", linkID).Scan(&total).Error; err != nil {
		return result, err
	}
	result.TotalPV = total.PV
	result.TotalUV = total.UV

	today := time.Now().Format("20060102")
	yesterdayDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	result.TodayPV = s.redisUint(ctx, fmt.Sprintf("stat:pv:%d:%s", linkID, today))
	result.TodayUV = s.redisPFCount(ctx, fmt.Sprintf("stat:uv:%d:%s", linkID, today))
	_ = s.db.WithContext(ctx).Table("stat_daily").Select("pv").Where("link_id = ? AND stat_date = ?", linkID, yesterdayDate).Scan(&result.YesterdayPV).Error
	result.TotalPV += result.TodayPV
	result.TotalUV += result.TodayUV
	return result, nil
}

func (s *StatService) Daily(ctx context.Context, linkID uint64, start, end time.Time) ([]DailyPoint, error) {
	var rows []struct {
		StatDate time.Time
		PV       uint64
		UV       uint64
	}
	if err := s.db.WithContext(ctx).Table("stat_daily").
		Select("stat_date, pv, uv").
		Where("link_id = ? AND stat_date BETWEEN ? AND ?", linkID, start.Format("2006-01-02"), end.Format("2006-01-02")).
		Order("stat_date ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	points := make([]DailyPoint, 0, len(rows)+1)
	for _, row := range rows {
		points = append(points, DailyPoint{Date: row.StatDate.Format("2006-01-02"), PV: row.PV, UV: row.UV})
	}
	if sameDay(end, time.Now()) {
		today := time.Now().Format("20060102")
		points = append(points, DailyPoint{
			Date: time.Now().Format("2006-01-02"),
			PV:   s.redisUint(ctx, fmt.Sprintf("stat:pv:%d:%s", linkID, today)),
			UV:   s.redisPFCount(ctx, fmt.Sprintf("stat:uv:%d:%s", linkID, today)),
		})
	}
	return points, nil
}

func (s *StatService) Hourly(ctx context.Context, linkID uint64, date time.Time) ([]HourlyPoint, error) {
	points := make([]HourlyPoint, 24)
	for hour := range points {
		points[hour] = HourlyPoint{Hour: uint8(hour)}
	}
	var rows []struct {
		StatHour uint8
		PV       uint64
	}
	if err := s.db.WithContext(ctx).Table("stat_hourly").
		Select("stat_hour, pv").
		Where("link_id = ? AND stat_date = ?", linkID, date.Format("2006-01-02")).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if int(row.StatHour) < len(points) {
			points[row.StatHour].PV = row.PV
		}
	}
	if sameDay(date, time.Now()) {
		day := time.Now().Format("20060102")
		for hour := 0; hour < 24; hour++ {
			points[hour].PV += s.redisUint(ctx, fmt.Sprintf("stat:hourly:%d:%s:%02d", linkID, day, hour))
		}
	}
	return points, nil
}

func (s *StatService) Geo(ctx context.Context, linkID uint64, start, end time.Time) ([]LabelValue, error) {
	var rows []LabelValue
	err := s.db.WithContext(ctx).Table("stat_geo").
		Select("IF(province = '', country, province) AS label, COALESCE(SUM(pv),0) AS value").
		Where("link_id = ? AND stat_date BETWEEN ? AND ?", linkID, start.Format("2006-01-02"), end.Format("2006-01-02")).
		Group("label").Order("value DESC").Limit(20).Scan(&rows).Error
	return rows, err
}

func (s *StatService) Device(ctx context.Context, linkID uint64, start, end time.Time) (DeviceStats, error) {
	var stats DeviceStats
	if err := s.aggregateLabel(ctx, "device", linkID, start, end, &stats.Device); err != nil {
		return stats, err
	}
	if err := s.aggregateLabel(ctx, "os", linkID, start, end, &stats.OS); err != nil {
		return stats, err
	}
	if err := s.aggregateLabel(ctx, "browser", linkID, start, end, &stats.Browser); err != nil {
		return stats, err
	}
	return stats, nil
}

func (s *StatService) aggregateLabel(ctx context.Context, column string, linkID uint64, start, end time.Time, dest *[]LabelValue) error {
	return s.db.WithContext(ctx).Table("stat_device").
		Select(column+" AS label, COALESCE(SUM(pv),0) AS value").
		Where("link_id = ? AND stat_date BETWEEN ? AND ?", linkID, start.Format("2006-01-02"), end.Format("2006-01-02")).
		Group(column).Order("value DESC").Limit(20).Scan(dest).Error
}

func (s *StatService) redisUint(ctx context.Context, key string) uint64 {
	value, err := s.redis.Get(ctx, key).Uint64()
	if err != nil {
		return 0
	}
	return value
}

func (s *StatService) redisPFCount(ctx context.Context, key string) uint64 {
	value, err := s.redis.PFCount(ctx, key).Result()
	if err != nil {
		return 0
	}
	return uint64(value)
}

func sameDay(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
