package service

import (
	"context"
	"gorm.io/gorm"
	"time"
)

type WindowStats struct {
	PV     uint64       `json:"pv"`
	UV     uint64       `json:"uv"`
	Daily  []DailyPoint `json:"daily"`
	Device DeviceStats  `json:"device"`
	Geo    GeoStats     `json:"geo"`
	Source SourceStats  `json:"source"`
}

func (s *StatService) windowLogs(ctx context.Context, linkID uint64, start, end time.Time) *gorm.DB {
	base := "link_id,visited_at,ip,device,os,browser,province,country,city,referer"
	liveCols := base + ",source_app"
	archiveCols := liveCols
	if !s.archiveSourceApp {
		archiveCols = base + ",'' AS source_app"
	}
	scope := ""
	args := []interface{}{start, end}
	if linkID > 0 {
		scope = " AND link_id = ?"
		args = append(args, linkID)
	}
	query := "SELECT " + liveCols + " FROM access_logs WHERE visited_at>=? AND visited_at<?" + scope
	query += " UNION ALL SELECT " + archiveCols + " FROM access_logs_archive WHERE visited_at>=? AND visited_at<?" + scope
	args = append(args, args...)
	return s.db.WithContext(ctx).Table("("+query+") AS visits", args...)
}

func (s *StatService) Window(ctx context.Context, linkID uint64, start, end time.Time) (WindowStats, error) {
	result := WindowStats{
		Daily:  []DailyPoint{},
		Geo:    GeoStats{Country: []LabelValue{}, Province: []LabelValue{}, City: []LabelValue{}},
		Device: DeviceStats{Device: []LabelValue{}, OS: []LabelValue{}, Browser: []LabelValue{}},
		Source: SourceStats{App: []LabelValue{}, Referer: []LabelValue{}},
	}
	var counts struct {
		PV uint64
		UV uint64
	}
	if err := s.windowLogs(ctx, linkID, start, end).Select("COUNT(*) AS pv,COUNT(DISTINCT ip) AS uv").Scan(&counts).Error; err != nil {
		return result, err
	}
	result.PV = counts.PV
	result.UV = counts.UV
	var daily []DailyPoint
	if err := s.windowLogs(ctx, linkID, start, end).Select("DATE_FORMAT(visited_at,'%Y-%m-%d') AS date,COUNT(*) AS pv,COUNT(DISTINCT ip) AS uv").Group("date").Order("date").Scan(&daily).Error; err != nil {
		return result, err
	}
	byDate := map[string]DailyPoint{}
	for _, p := range daily {
		byDate[p.Date] = p
	}
	for d := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location()); d.Before(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		p := byDate[key]
		p.Date = key
		result.Daily = append(result.Daily, p)
	}
	for _, dim := range []struct {
		expr string
		dest *[]LabelValue
	}{{"COALESCE(NULLIF(device,''),'unknown')", &result.Device.Device}, {"COALESCE(NULLIF(os,''),'未知')", &result.Device.OS}, {"COALESCE(NULLIF(browser,''),'未知')", &result.Device.Browser}, {"COALESCE(NULLIF(country,''),'未知')", &result.Geo.Country}} {
		if err := s.windowLogs(ctx, linkID, start, end).Select(dim.expr + " AS label,COUNT(*) AS value").Group("label").Order("value DESC,label").Scan(dim.dest).Error; err != nil {
			return result, err
		}
	}
	if err := s.windowLogs(ctx, linkID, start, end).
		Where("country = '中国' AND province <> ''").
		Select("province AS label,COUNT(*) AS value").
		Group("label").Order("value DESC,label").
		Scan(&result.Geo.Province).Error; err != nil {
		return result, err
	}
	// 国内城市 TOP（有城市名时）
	if err := s.windowLogs(ctx, linkID, start, end).
		Where("country = '中国' AND city <> ''").
		Select("city AS label,COUNT(*) AS value").
		Group("label").Order("value DESC,label").Limit(20).
		Scan(&result.Geo.City).Error; err != nil {
		return result, err
	}
	// 来源 APP（微信/抖音等，由 UA+Referer 解析）
	if err := s.windowLogs(ctx, linkID, start, end).
		Select("COALESCE(NULLIF(source_app,''),'直接访问') AS label,COUNT(*) AS value").
		Group("label").Order("value DESC,label").
		Scan(&result.Source.App).Error; err != nil {
		return result, err
	}
	// Referer 域名（空视为直接访问）
	if err := s.windowLogs(ctx, linkID, start, end).
		Select(`COALESCE(NULLIF(SUBSTRING_INDEX(SUBSTRING_INDEX(referer, '://', -1), '/', 1), ''), '直接访问') AS label, COUNT(*) AS value`).
		Group("label").Order("value DESC,label").Limit(20).
		Scan(&result.Source.Referer).Error; err != nil {
		return result, err
	}
	return result, nil
}

// RollingHourly returns 24 complete one-hour intervals ending at the supplied time.
func (s *StatService) RollingHourly(ctx context.Context, linkID uint64, end time.Time) ([]HourlyPoint, error) {
	end = end.Truncate(time.Second)
	start := end.Add(-24 * time.Hour)
	var counts []struct {
		Bucket int
		PV     uint64
	}
	if err := s.windowLogs(ctx, linkID, start, end).Select("FLOOR(TIMESTAMPDIFF(SECOND, ?, visited_at)/3600) AS bucket,COUNT(*) AS pv", start).Group("bucket").Scan(&counts).Error; err != nil {
		return nil, err
	}
	result := make([]HourlyPoint, 24)
	for i := range result {
		t := start.Add(time.Duration(i) * time.Hour)
		result[i] = HourlyPoint{Hour: uint8(t.Hour()), Start: t.Format(time.RFC3339), End: t.Add(time.Hour).Format(time.RFC3339)}
	}
	for _, c := range counts {
		if c.Bucket >= 0 && c.Bucket < 24 {
			result[c.Bucket].PV = c.PV
		}
	}
	return result, nil
}
