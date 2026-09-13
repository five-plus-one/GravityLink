package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type StatService struct {
	db               *gorm.DB
	redis            *redis.Client
	archiveSourceApp bool
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
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
	Hour  uint8  `json:"hour"`
	PV    uint64 `json:"pv"`
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
	return &StatService{db: db, redis: redis, archiveSourceApp: db.Migrator().HasColumn("access_logs_archive", "source_app")}
}

func (s *StatService) Summary(ctx context.Context, linkID uint64) (SummaryStats, error) {
	var result SummaryStats
	type totals struct {
		PV uint64
		UV uint64
	}
	var total totals
	if err := s.db.WithContext(ctx).Table("stat_daily").Select("COALESCE(SUM(pv),0) AS pv, COALESCE(SUM(uv),0) AS uv").Where("link_id = ? AND stat_date < ?", linkID, time.Now().Format("2006-01-02")).Scan(&total).Error; err != nil {
		return result, err
	}
	result.TotalPV = total.PV
	result.TotalUV = total.UV

	// 今日：从 access_logs 实时计数，与访客记录页保持一致
	today := time.Now().Format("2006-01-02")
	var todayPV int64
	s.db.WithContext(ctx).Table("access_logs").Where("link_id = ? AND visited_at >= ?", linkID, today).Count(&todayPV)
	result.TodayPV = uint64(todayPV)

	var todayIP int64
	s.db.WithContext(ctx).Table("access_logs").Where("link_id = ? AND visited_at >= ?", linkID, today).Distinct("ip").Count(&todayIP)
	result.TodayUV = uint64(todayIP)

	yesterdayDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_ = s.db.WithContext(ctx).Table("stat_daily").Select("pv").Where("link_id = ? AND stat_date = ?", linkID, yesterdayDate).Scan(&result.YesterdayPV).Error
	result.TotalPV += result.TodayPV
	result.TotalUV += result.TodayUV
	return result, nil
}

// SummaryAll 返回全部链接的聚合统计。
func (s *StatService) SummaryAll(ctx context.Context) (SummaryStats, error) {
	var result SummaryStats
	type totals struct {
		PV uint64
		UV uint64
	}
	var total totals
	if err := s.db.WithContext(ctx).Table("stat_daily").Select("COALESCE(SUM(pv),0) AS pv, COALESCE(SUM(uv),0) AS uv").Where("stat_date < ?", time.Now().Format("2006-01-02")).Scan(&total).Error; err != nil {
		return result, err
	}
	result.TotalPV = total.PV
	result.TotalUV = total.UV

	// 今日：用 access_logs 实时计数（PV）；UV 用 distinct IP 估算
	today := time.Now().Format("2006-01-02")
	var todayPV int64
	s.db.WithContext(ctx).Table("access_logs").Where("visited_at >= ?", today).Count(&todayPV)
	result.TodayPV = uint64(todayPV)

	var todayIP int64
	s.db.WithContext(ctx).Table("access_logs").Where("visited_at >= ?", today).Distinct("ip").Count(&todayIP)
	result.TodayUV = uint64(todayIP)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_ = s.db.WithContext(ctx).Table("stat_daily").Select("COALESCE(SUM(pv),0)").Where("stat_date = ?", yesterday).Scan(&result.YesterdayPV).Error
	result.TotalPV += result.TodayPV
	result.TotalUV += result.TodayUV
	return result, nil
}

// DailyAll 返回全部链接的按天聚合统计。
func (s *StatService) DailyAll(ctx context.Context, start, end time.Time) ([]DailyPoint, error) {
	var rows []struct {
		StatDate time.Time
		PV       uint64
		UV       uint64
	}
	if err := s.db.WithContext(ctx).Table("stat_daily").
		Select("stat_date, SUM(pv) AS pv, SUM(uv) AS uv").
		Where("stat_date BETWEEN ? AND ?", start.Format("2006-01-02"), end.Format("2006-01-02")).
		Where("stat_date < ?", time.Now().Format("2006-01-02")).Group("stat_date").
		Where("stat_date < ?", time.Now().Format("2006-01-02")).Order("stat_date ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	points := make([]DailyPoint, 0, len(rows)+1)
	for _, row := range rows {
		points = append(points, DailyPoint{Date: row.StatDate.Format("2006-01-02"), PV: row.PV, UV: row.UV})
	}
	if sameDay(end, time.Now()) {
		today := time.Now().Format("2006-01-02")
		var todayPV int64
		s.db.WithContext(ctx).Table("access_logs").Where("visited_at >= ?", today).Count(&todayPV)
		var todayIP int64
		s.db.WithContext(ctx).Table("access_logs").Where("visited_at >= ?", today).Distinct("ip").Count(&todayIP)
		points = append(points, DailyPoint{
			Date: time.Now().Format("2006-01-02"),
			PV:   uint64(todayPV),
			UV:   uint64(todayIP),
		})
	}
	return points, nil
}

// HourlyAll 返回全部链接的按小时聚合统计。
func (s *StatService) HourlyAll(ctx context.Context, date time.Time) ([]HourlyPoint, error) {
	points := make([]HourlyPoint, 24)
	for hour := range points {
		points[hour] = HourlyPoint{Hour: uint8(hour)}
	}
	var rows []struct {
		StatHour uint8
		PV       uint64
	}
	if err := s.db.WithContext(ctx).Table("stat_hourly").
		Select("stat_hour, SUM(pv) AS pv").
		Where("stat_date = ?", date.Format("2006-01-02")).
		Group("stat_hour").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if int(row.StatHour) < len(points) {
			points[row.StatHour].PV = row.PV
		}
	}
	if sameDay(date, time.Now()) {
		// 今日实时：从 access_logs 按小时聚合
		today := time.Now().Format("2006-01-02")
		var hourRows []struct {
			Hr uint8
			PV int64
		}
		s.db.WithContext(ctx).Table("access_logs").
			Select("HOUR(visited_at) AS hr, COUNT(*) AS pv").
			Where("visited_at >= ?", today).
			Group("HOUR(visited_at)").
			Scan(&hourRows)
		for _, r := range hourRows {
			if int(r.Hr) < len(points) {
				points[r.Hr].PV += uint64(r.PV)
			}
		}
	}
	return points, nil
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
		Where("stat_date < ?", time.Now().Format("2006-01-02")).Order("stat_date ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	points := make([]DailyPoint, 0, len(rows)+1)
	for _, row := range rows {
		points = append(points, DailyPoint{Date: row.StatDate.Format("2006-01-02"), PV: row.PV, UV: row.UV})
	}
	if sameDay(end, time.Now()) {
		today := time.Now().Format("2006-01-02")
		var todayPV int64
		s.db.WithContext(ctx).Table("access_logs").Where("link_id = ? AND visited_at >= ?", linkID, today).Count(&todayPV)
		var todayIP int64
		s.db.WithContext(ctx).Table("access_logs").Where("link_id = ? AND visited_at >= ?", linkID, today).Distinct("ip").Count(&todayIP)
		points = append(points, DailyPoint{
			Date: time.Now().Format("2006-01-02"),
			PV:   uint64(todayPV),
			UV:   uint64(todayIP),
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
		today := time.Now().Format("2006-01-02")
		var hourRows []struct {
			Hr uint8
			PV int64
		}
		s.db.WithContext(ctx).Table("access_logs").
			Select("HOUR(visited_at) AS hr, COUNT(*) AS pv").
			Where("link_id = ? AND visited_at >= ?", linkID, today).
			Group("HOUR(visited_at)").
			Scan(&hourRows)
		for _, r := range hourRows {
			if int(r.Hr) < len(points) {
				points[r.Hr].PV += uint64(r.PV)
			}
		}
	}
	return points, nil
}

func (s *StatService) Geo(ctx context.Context, linkID uint64, start, end time.Time) ([]LabelValue, error) {
	rows := []LabelValue{}
	err := s.distribution(ctx, "stat_geo", "COALESCE(NULLIF(province, ''), NULLIF(country, ''), '未知')", linkID, start, end, &rows)
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
	*dest = []LabelValue{}
	return s.distribution(ctx, "stat_device", "COALESCE(NULLIF("+column+", ''), '未知')", linkID, start, end, dest)
}

// Query disjoint historical aggregates and today's persisted visits.
func (s *StatService) distribution(ctx context.Context, table, label string, linkID uint64, start, end time.Time, dest *[]LabelValue) error {
	today := time.Now().Format("2006-01-02")
	scope := ""
	args := []any{start.Format("2006-01-02"), end.Format("2006-01-02"), today}
	if linkID > 0 {
		scope = " AND link_id = ?"
		args = append(args, linkID)
	}
	sql := "SELECT " + label + " AS label, SUM(pv) AS value FROM " + table + " WHERE stat_date BETWEEN ? AND ? AND stat_date < ?" + scope + " GROUP BY label"
	if start.Format("2006-01-02") <= today && end.Format("2006-01-02") >= today {
		sql += " UNION ALL SELECT " + label + " AS label, COUNT(*) AS value FROM access_logs WHERE visited_at >= ? AND visited_at < ?" + scope + " GROUP BY label"
		args = append(args, today, time.Now().AddDate(0, 0, 1).Format("2006-01-02"))
		if linkID > 0 {
			args = append(args, linkID)
		}
	}
	return s.db.WithContext(ctx).Raw("SELECT label, SUM(value) AS value FROM ("+sql+") AS distribution GROUP BY label ORDER BY value DESC, label ASC", args...).Scan(dest).Error
}

func (s *StatService) HourlyRange(ctx context.Context, linkID uint64, start, end time.Time) ([]HourlyPoint, error) {
	today := time.Now().Format("2006-01-02")
	scope := ""
	args := []any{start.Format("2006-01-02"), end.Format("2006-01-02"), today}
	if linkID > 0 {
		scope = " AND link_id = ?"
		args = append(args, linkID)
	}
	sql := "SELECT stat_hour AS hour, SUM(pv) AS pv FROM stat_hourly WHERE stat_date BETWEEN ? AND ? AND stat_date < ?" + scope + " GROUP BY stat_hour"
	if start.Format("2006-01-02") <= today && end.Format("2006-01-02") >= today {
		sql += " UNION ALL SELECT HOUR(visited_at) AS hour, COUNT(*) AS pv FROM access_logs WHERE visited_at >= ? AND visited_at < ?" + scope + " GROUP BY HOUR(visited_at)"
		args = append(args, today, time.Now().AddDate(0, 0, 1).Format("2006-01-02"))
		if linkID > 0 {
			args = append(args, linkID)
		}
	}
	var rows []HourlyPoint
	err := s.db.WithContext(ctx).Raw("SELECT hour, SUM(pv) AS pv FROM ("+sql+") AS hours GROUP BY hour", args...).Scan(&rows).Error
	points := make([]HourlyPoint, 24)
	for h := range points {
		points[h].Hour = uint8(h)
	}
	for _, row := range rows {
		if row.Hour < 24 {
			points[row.Hour] = row
		}
	}
	return points, err
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

// Reset 清空某链接的全部统计数据：Redis 实时计数 + MySQL 聚合表。
// access_logs 原始日志保留（审计需要），仅统计口径归零。
func (s *StatService) Reset(ctx context.Context, linkID uint64) error {
	if linkID == 0 {
		return fmt.Errorf("invalid link id")
	}
	// Redis：SCAN 该链接的 stat:pv/uv/hourly/dev/geo keys 后删除
	var cursor uint64
	for {
		batch, next, err := s.redis.Scan(ctx, cursor, fmt.Sprintf("stat:*:%d:*", linkID), 200).Result()
		if err != nil {
			return err
		}
		if len(batch) > 0 {
			if err := s.redis.Del(ctx, batch...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	// MySQL：清空聚合表对应行
	for _, table := range []string{"stat_daily", "stat_hourly", "stat_device", "stat_geo"} {
		if err := s.db.WithContext(ctx).Exec(fmt.Sprintf("DELETE FROM %s WHERE link_id = ?", table), linkID).Error; err != nil {
			return err
		}
	}
	return nil
}

func sameDay(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// VisitorLog 访客记录查询结果。
type VisitorLog struct {
	ID        uint64    `json:"id"`
	LinkID    uint64    `json:"link_id"`
	LinkCode  string    `json:"link_code"`
	LinkTitle string    `json:"link_title"`
	VisitedAt time.Time `json:"visited_at"`
	IP        string    `json:"ip"`
	Country   string    `json:"country"`
	Province  string    `json:"province"`
	City      string    `json:"city"`
	Device    string    `json:"device"`
	OS        string    `json:"os"`
	Browser   string    `json:"browser"`
	Referer   string    `json:"referer"`
	SourceApp string    `json:"source_app"`
}

type VisitorLogQuery struct {
	LinkID  uint64
	Start   time.Time
	End     time.Time
	Keyword string
	Limit   int
	Offset  int
}

type VisitorLogResult struct {
	Items []VisitorLog `json:"items"`
	Total int64        `json:"total"`
}

// ListVisitors 查询访客记录，支持按链接、时间范围、关键词筛选。
func (s *StatService) ListVisitors(ctx context.Context, q VisitorLogQuery) (VisitorLogResult, error) {
	result := VisitorLogResult{Items: []VisitorLog{}}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}

	columns := "id, link_id, visited_at, ip, country, province, city, device, os, browser, referer"
	archiveSource := "'' AS source_app"
	if s.archiveSourceApp {
		archiveSource = "source_app"
	}
	source := "(SELECT " + columns + ", source_app FROM access_logs UNION ALL SELECT " + columns + ", " + archiveSource + " FROM access_logs_archive) AS access_logs"

	query := s.db.WithContext(ctx).Table(source).
		Select("access_logs.id, access_logs.link_id, links.code AS link_code, COALESCE(links.title, '') AS link_title, access_logs.visited_at, access_logs.ip, COALESCE(access_logs.country,'') AS country, COALESCE(access_logs.province,'') AS province, COALESCE(access_logs.city,'') AS city, access_logs.device, COALESCE(access_logs.os,'') AS os, COALESCE(access_logs.browser,'') AS browser, COALESCE(access_logs.referer,'') AS referer, COALESCE(access_logs.source_app,'') AS source_app").
		Joins("LEFT JOIN links ON links.id = access_logs.link_id AND links.deleted_at IS NULL")

	if q.LinkID > 0 {
		query = query.Where("access_logs.link_id = ?", q.LinkID)
	}
	if !q.Start.IsZero() {
		query = query.Where("access_logs.visited_at >= ?", q.Start)
	}
	if !q.End.IsZero() {
		query = query.Where("access_logs.visited_at < ?", q.End)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("access_logs.ip LIKE ? OR links.code LIKE ? OR COALESCE(links.title,'') LIKE ?", kw, kw, kw)
	}

	if err := query.Count(&result.Total).Error; err != nil {
		return result, err
	}

	err := query.Order("access_logs.visited_at DESC, access_logs.id DESC").
		Limit(q.Limit).Offset(q.Offset).
		Scan(&result.Items).Error
	return result, err
}
