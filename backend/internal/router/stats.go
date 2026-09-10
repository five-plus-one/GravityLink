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
	// 访客记录查询
	group.GET("/stats/visitors", func(c *gin.Context) {
		q := service.VisitorLogQuery{
			Keyword: c.Query("keyword"),
		}
		if v := c.Query("link_id"); v != "" {
			q.LinkID, _ = strconv.ParseUint(v, 10, 64)
		}
		if v := c.Query("start"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				q.Start = t
			}
		}
		if v := c.Query("end"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				q.End = t.Add(24*time.Hour - time.Second)
			}
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
		start, end := parseDateRange(c)
		data, err := stats.DailyAll(c.Request.Context(), start, end)
		writeStatResult(c, data, err)
	})
	group.GET("/stats/overview/hourly", func(c *gin.Context) {
		date := parseDate(c.DefaultQuery("date", time.Now().Format("2006-01-02")), time.Now())
		data, err := stats.HourlyAll(c.Request.Context(), date)
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
		start, end := parseDateRange(c)
		data, err := stats.Daily(c.Request.Context(), linkID, start, end)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/:link_id/hourly", func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		date := parseDate(c.DefaultQuery("date", time.Now().Format("2006-01-02")), time.Now())
		data, err := stats.Hourly(c.Request.Context(), linkID, date)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/:link_id/geo", func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		start, end := parseDateRange(c)
		data, err := stats.Geo(c.Request.Context(), linkID, start, end)
		writeStatResult(c, data, err)
	})

	group.GET("/stats/:link_id/device", func(c *gin.Context) {
		linkID, ok := parseLinkID(c)
		if !ok {
			return
		}
		start, end := parseDateRange(c)
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

func parseDateRange(c *gin.Context) (time.Time, time.Time) {
	end := parseDate(c.Query("end"), time.Now())
	start := parseDate(c.Query("start"), end.AddDate(0, 0, -30))
	if end.Sub(start) > 90*24*time.Hour {
		start = end.AddDate(0, 0, -90)
	}
	return start, end
}

func parseDate(value string, fallback time.Time) time.Time {
	if value == "" {
		return fallback
	}
	parsed, err := time.Parse("2006-01-02", value)
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
