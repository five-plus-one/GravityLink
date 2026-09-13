package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerStatRoutes(group *gin.RouterGroup, stats *service.StatService) {
	group.GET("/stats/window", func(c *gin.Context) {
		start, e1 := parseVisitorTime(c.Query("start"), false)
		end, e2 := parseVisitorTime(c.Query("end"), true)
		if e1 != nil || e2 != nil || start.IsZero() || end.IsZero() || !start.Before(end) || end.Sub(start) > 90*24*time.Hour {
			response.Error(c, 400, 4202, "请选择跨度不超过90天的有效时段")
			return
		}
		id, err := strconv.ParseUint(c.DefaultQuery("link_id", "0"), 10, 64)
		if err != nil {
			response.Error(c, 400, 4202, "请选择有效链接")
			return
		}
		data, err := stats.Window(c.Request.Context(), id, start, end)
		writeStatResult(c, data, err)
	})
	// 访客记录查询
	group.GET("/stats/visitors", func(c *gin.Context) {
		q := service.VisitorLogQuery{
			Keyword: c.Query("keyword"),
		}
		if v := c.Query("link_id"); v != "" {
			q.LinkID, _ = strconv.ParseUint(v, 10, 64)
		}
		var err error
		q.Start, err = parseVisitorTime(c.Query("start"), false)
		if err == nil {
			q.End, err = parseVisitorTime(c.Query("end"), true)
		}
		if err != nil || (!q.Start.IsZero() && !q.End.IsZero() && !q.Start.Before(q.End)) {
			response.Error(c, 400, 4202, "请选择有效的开始和结束时间")
			return
		}
		q.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
		q.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
		data, err := stats.ListVisitors(c.Request.Context(), q)
		writeStatResult(c, data, err)
	})

	// 全局聚合统计（概览页用）
	group.GET("/stats/overview/summary", func(c *gin.Context) {
		data, err := stats.SummaryAll(c.Request.Context())
		writeStatResult(c, data, err)
	})
	group.GET("/stats/overview/daily", func(c *gin.Context) {
		start, end, valid := parseDateRange(c)
		if !valid {
			return
		}
		data, err := stats.DailyAll(c.Request.Context(), start, end)
		writeStatResult(c, data, err)
	})
	group.GET("/stats/overview/hourly", func(c *gin.Context) {
		if c.Query("date") == "" && c.Query("start") == "" && c.Query("end") == "" {
			id, _ := strconv.ParseUint(c.Param("link_id"), 10, 64)
			data, err := stats.RollingHourly(c.Request.Context(), id, time.Now())
			writeStatResult(c, data, err)
			return
		}
		date := parseDate(c.DefaultQuery("date", time.Now().Format("2006-01-02")), time.Now())
		start, end := date, date
		if c.Query("start") != "" || c.Query("end") != "" {
			var valid bool
			start, end, valid = parseDateRange(c)
			if !valid {
				return
			}
		}
		data, err := stats.HourlyRange(c.Request.Context(), 0, start, end)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/overview/geo", func(c *gin.Context) {
		start, end, valid := parseDateRange(c)
		if !valid {
			return
		}
		data, err := stats.Geo(c.Request.Context(), 0, start, end)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/overview/device", func(c *gin.Context) {
		start, end, valid := parseDateRange(c)
		if !valid {
			return
		}
		data, err := stats.Device(c.Request.Context(), 0, start, end)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/:link_id/summary", func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		data, err := stats.Summary(c.Request.Context(), linkID)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/:link_id/daily", func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		start, end, valid := parseDateRange(c)
		if !valid {
			return
		}
		data, err := stats.Daily(c.Request.Context(), linkID, start, end)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/:link_id/hourly", func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		if c.Query("date") == "" && c.Query("start") == "" && c.Query("end") == "" {
			id, _ := strconv.ParseUint(c.Param("link_id"), 10, 64)
			data, err := stats.RollingHourly(c.Request.Context(), id, time.Now())
			writeStatResult(c, data, err)
			return
		}
		date := parseDate(c.DefaultQuery("date", time.Now().Format("2006-01-02")), time.Now())
		start, end := date, date
		if c.Query("start") != "" || c.Query("end") != "" {
			var valid bool
			start, end, valid = parseDateRange(c)
			if !valid {
				return
			}
		}
		data, err := stats.HourlyRange(c.Request.Context(), linkID, start, end)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/:link_id/geo", func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		start, end, valid := parseDateRange(c)
		if !valid {
			return
		}
		data, err := stats.Geo(c.Request.Context(), linkID, start, end)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/:link_id/device", func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		start, end, valid := parseDateRange(c)
		if !valid {
			return
		}
		data, err := stats.Device(c.Request.Context(), linkID, start, end)
		writeStatResult(c, data, err)
	})

	// P1：重置链接统计（危险操作，管理员专用；access_logs 原始日志保留）
	group.POST("/stats/:link_id/reset", middleware.RequireRole("admin"), func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		if err := stats.Reset(c.Request.Context(), linkID); err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "重置统计失败")
			return
		}
		writeStatResult(c, gin.H{"reset": true}, nil)
	})
}

func parseLinkID(c *gin.Context) (uint64, bool) {
	linkID, err := strconv.ParseUint(c.Param("link_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4201, "invalid link id")
		return 0, false
	}
	return linkID, true
}

func parseDateRange(c *gin.Context) (time.Time, time.Time, bool) {
	end := parseDate(c.Query("end"), time.Now())
	start := parseDate(c.Query("start"), end.AddDate(0, 0, -29))
	for _, key := range []string{"start", "end"} {
		if value := c.Query(key); value != "" {
			if _, err := time.ParseInLocation("2006-01-02", value, time.Local); err != nil {
				response.Error(c, 400, 4202, "请选择有效日期")
				return start, end, false
			}
		}
	}
	if start.After(end) || end.Sub(start) > 90*24*time.Hour {
		response.Error(c, 400, 4202, "请选择开始早于结束、跨度不超过90天的日期范围")
		return start, end, false
	}
	return start, end, true
}

func parseDate(value string, fallback time.Time) time.Time {
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return fallback
	}
	return parsed
}

func writeStatResult(c *gin.Context, data interface{}, err error) {
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 5000, "query stats failed")
		return
	}
	response.OK(c, data)
}

func parseVisitorTime(value string, end bool) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if len(value) == 10 {
		t, err := time.ParseInLocation("2006-01-02", value, time.Local)
		if end {
			t = t.AddDate(0, 0, 1)
		}
		return t, err
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	return t.In(time.Local), err
}
