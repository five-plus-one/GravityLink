package worker

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/service"
)

const (
	statPrefixPV     = "stat:pv:"
	statPrefixUV     = "stat:uv:"
	statPrefixHourly = "stat:hourly:"
)

// StatFlusher 周期性把 Redis 中的访问计数（非今日）落盘到 MySQL 聚合表：
// stat:pv/uv:{link}:{day} → stat_daily，stat:hourly:{link}:{day}:{hour} → stat_hourly，
// stat:dev:{link}:{day} → stat_device，stat:geo:{link}:{day} → stat_geo。
// 今日 key 保留在 Redis，供读接口叠加实时值。
//
// 写入采用幂等覆盖（写成功后才删 key），单实例部署下重试安全。
type StatFlusher struct {
	db     *gorm.DB
	redis  *redis.Client
	logger *slog.Logger
}

func NewStatFlusher(db *gorm.DB, redis *redis.Client, logger *slog.Logger) *StatFlusher {
	return &StatFlusher{db: db, redis: redis, logger: logger}
}

func (f *StatFlusher) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.flush(ctx)
		}
	}
}

func (f *StatFlusher) flush(ctx context.Context) {
	today := time.Now().Format("20060102")
	if err := f.flushDaily(ctx, today); err != nil {
		f.logger.Warn("flush stat_daily failed", "error", err)
	}
	if err := f.flushHourly(ctx, today); err != nil {
		f.logger.Warn("flush stat_hourly failed", "error", err)
	}
	if err := f.flushDevice(ctx, today); err != nil {
		f.logger.Warn("flush stat_device failed", "error", err)
	}
	if err := f.flushGeo(ctx, today); err != nil {
		f.logger.Warn("flush stat_geo failed", "error", err)
	}
}

// flushDaily 把过期的 stat:pv / stat:uv key 写入 stat_daily（uv 同时写入 ip_count）。
func (f *StatFlusher) flushDaily(ctx context.Context, today string) error {
	targets := make(map[string]struct {
		linkID uint64
		day    string
	})
	for _, pattern := range []string{statPrefixPV + "*", statPrefixUV + "*"} {
		for _, key := range f.scanKeys(ctx, pattern) {
			body := strings.TrimPrefix(strings.TrimPrefix(key, statPrefixPV), statPrefixUV)
			linkID, day, ok := splitDayBody(body)
			if !ok {
				f.logger.Warn("skip invalid stat key", "key", key)
				continue
			}
			targets[body] = struct {
				linkID uint64
				day    string
			}{linkID: linkID, day: day}
		}
	}

	for body, target := range targets {
		if target.day >= today {
			continue
		}
		pv, _ := f.redis.Get(ctx, statPrefixPV+body).Uint64()
		uv, _ := f.redis.PFCount(ctx, statPrefixUV+body).Result()

		if pv == 0 && uv == 0 {
			_ = f.redis.Del(ctx, statPrefixPV+body, statPrefixUV+body).Err()
			continue
		}

		statDate, err := parseStatDate(target.day)
		if err != nil {
			f.logger.Warn("skip invalid stat date", "day", target.day, "error", err)
			continue
		}
		row := model.StatDaily{LinkID: target.linkID, StatDate: statDate, PV: pv, UV: uint64(uv), IPCount: uint64(uv)}
		if err := f.db.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{"pv", "uv", "ip_count"}),
		}).Create(&row).Error; err != nil {
			return err
		}
		_ = f.redis.Del(ctx, statPrefixPV+body, statPrefixUV+body).Err()
	}
	return nil
}

// flushHourly 把过期的 stat:hourly key 写入 stat_hourly。
func (f *StatFlusher) flushHourly(ctx context.Context, today string) error {
	for _, key := range f.scanKeys(ctx, statPrefixHourly+"*") {
		body := strings.TrimPrefix(key, statPrefixHourly)
		parts := strings.Split(body, ":")
		if len(parts) != 3 {
			f.logger.Warn("skip invalid stat key", "key", key)
			continue
		}
		linkID, day, ok := splitDayBody(parts[0] + ":" + parts[1])
		if !ok || day >= today {
			continue
		}
		hour64, err := strconv.ParseUint(parts[2], 10, 8)
		if err != nil || hour64 > 23 {
			f.logger.Warn("skip invalid stat hour", "key", key)
			continue
		}

		pv, _ := f.redis.Get(ctx, key).Uint64()
		if pv == 0 {
			_ = f.redis.Del(ctx, key).Err()
			continue
		}

		statDate, err := parseStatDate(day)
		if err != nil {
			f.logger.Warn("skip invalid stat date", "day", day, "error", err)
			continue
		}
		row := model.StatHourly{LinkID: linkID, StatDate: statDate, StatHour: uint8(hour64), PV: pv}
		if err := f.db.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{"pv"}),
		}).Create(&row).Error; err != nil {
			return err
		}
		_ = f.redis.Del(ctx, key).Err()
	}
	return nil
}

// flushDevice 把过期的 stat:dev Hash 写入 stat_device。
func (f *StatFlusher) flushDevice(ctx context.Context, today string) error {
	for _, key := range f.scanKeys(ctx, service.DeviceStatKeyPrefix+"*") {
		body := strings.TrimPrefix(key, service.DeviceStatKeyPrefix)
		linkID, day, ok := splitDayBody(body)
		if !ok || day >= today {
			continue
		}

		fields, err := f.redis.HGetAll(ctx, key).Result()
		if err != nil {
			return err
		}
		if len(fields) == 0 {
			_ = f.redis.Del(ctx, key).Err()
			continue
		}

		statDate, err := parseStatDate(day)
		if err != nil {
			f.logger.Warn("skip invalid stat date", "day", day, "error", err)
			continue
		}

		rows := make([]model.StatDevice, 0, len(fields))
		for field, pv := range fields {
			device, osName, browser, ok := splitDeviceField(field)
			if !ok {
				f.logger.Warn("skip invalid device field", "key", key, "field", field)
				continue
			}
			pv64, err := strconv.ParseUint(pv, 10, 64)
			if err != nil || pv64 == 0 {
				continue
			}
			rows = append(rows, model.StatDevice{
				LinkID: linkID, StatDate: statDate,
				Device: device, OS: osName, Browser: browser, PV: pv64,
			})
		}
		if len(rows) == 0 {
			_ = f.redis.Del(ctx, key).Err()
			continue
		}

		if err := f.db.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{"pv"}),
		}).CreateInBatches(rows, 100).Error; err != nil {
			return err
		}
		_ = f.redis.Del(ctx, key).Err()
	}
	return nil
}

// flushGeo 把过期的 stat:geo Hash 写入 stat_geo。
func (f *StatFlusher) flushGeo(ctx context.Context, today string) error {
	for _, key := range f.scanKeys(ctx, service.GeoStatKeyPrefix+"*") {
		body := strings.TrimPrefix(key, service.GeoStatKeyPrefix)
		linkID, day, ok := splitDayBody(body)
		if !ok || day >= today {
			continue
		}

		fields, err := f.redis.HGetAll(ctx, key).Result()
		if err != nil {
			return err
		}
		if len(fields) == 0 {
			_ = f.redis.Del(ctx, key).Err()
			continue
		}

		statDate, err := parseStatDate(day)
		if err != nil {
			f.logger.Warn("skip invalid stat date", "day", day, "error", err)
			continue
		}

		rows := make([]model.StatGeo, 0, len(fields))
		for field, pv := range fields {
			country, province, ok := splitGeoField(field)
			if !ok {
				f.logger.Warn("skip invalid geo field", "key", key, "field", field)
				continue
			}
			pv64, err := strconv.ParseUint(pv, 10, 64)
			if err != nil || pv64 == 0 {
				continue
			}
			rows = append(rows, model.StatGeo{
				LinkID: linkID, StatDate: statDate,
				Country: country, Province: province, PV: pv64,
			})
		}
		if len(rows) == 0 {
			_ = f.redis.Del(ctx, key).Err()
			continue
		}

		if err := f.db.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{"pv"}),
		}).CreateInBatches(rows, 100).Error; err != nil {
			return err
		}
		_ = f.redis.Del(ctx, key).Err()
	}
	return nil
}

func (f *StatFlusher) scanKeys(ctx context.Context, pattern string) []string {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := f.redis.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			f.logger.Warn("scan stat keys failed", "pattern", pattern, "error", err)
			return keys
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			return keys
		}
	}
}

// splitDayBody 解析 "{linkID}:{yyyymmdd}" 主体。
func splitDayBody(body string) (linkID uint64, day string, ok bool) {
	parts := strings.Split(body, ":")
	if len(parts) != 2 || len(parts[1]) != 8 {
		return 0, "", false
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	if _, err := time.ParseInLocation("20060102", parts[1], time.Local); err != nil {
		return 0, "", false
	}
	return id, parts[1], true
}

// splitDeviceField 解析 Hash field "device|os|browser"。
func splitDeviceField(field string) (device, osName, browser string, ok bool) {
	parts := strings.SplitN(field, "|", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// splitGeoField 解析 Hash field "country|province"。
func splitGeoField(field string) (country, province string, ok bool) {
	parts := strings.SplitN(field, "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// parseStatDate 把 yyyymmdd 解析为本地时区日期（存入 DATE 列）。
func parseStatDate(day string) (time.Time, error) {
	return time.ParseInLocation("20060102", day, time.Local)
}
