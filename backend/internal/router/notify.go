package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerNotifyRoutes(admin *gin.RouterGroup, notifier *service.Notifier) {
	admin.GET("/notify/channels", func(c *gin.Context) {
		items, err := notifier.ListChannels(c.Request.Context())
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "读取通知渠道失败")
			return
		}
		response.OK(c, gin.H{"items": items, "has_enabled": notifier.HasEnabledChannel(c.Request.Context())})
	})

	admin.PUT("/notify/channels", func(c *gin.Context) {
		var input service.SaveChannelsInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4001, "请求格式无效")
			return
		}
		items, err := notifier.SaveChannels(c.Request.Context(), input)
		if err != nil {
			msg := err.Error()
			if errors.Is(err, service.ErrNotifySecret) {
				response.Error(c, 400, 4001, msg)
				return
			}
			if errors.Is(err, service.ErrNotify) {
				response.Error(c, 400, 4001, strings.TrimPrefix(msg, service.ErrNotify.Error()+": "))
				return
			}
			response.Error(c, 500, 5000, "保存通知渠道失败")
			return
		}
		response.OK(c, gin.H{"items": items, "has_enabled": notifier.HasEnabledChannel(c.Request.Context())})
	})

	admin.POST("/notify/channels/:type/test", func(c *gin.Context) {
		channelType := c.Param("type")
		if err := notifier.TestChannel(c.Request.Context(), channelType); err != nil {
			msg := err.Error()
			if errors.Is(err, service.ErrNotify) || errors.Is(err, service.ErrNotifySecret) {
				response.Error(c, 400, 4001, strings.TrimPrefix(msg, service.ErrNotify.Error()+": "))
				return
			}
			// SMTP/网络错误：中文摘要，不透出内部细节
			response.Error(c, 400, 4001, "测试发送失败："+msg)
			return
		}
		response.OK(c, gin.H{"ok": true, "message": "测试通知已发送，请检查目标渠道"})
	})

	admin.GET("/notify/events", func(c *gin.Context) {
		cfg, err := notifier.LoadEvents(c.Request.Context())
		if err != nil {
			response.Error(c, 500, 5000, "读取事件配置失败")
			return
		}
		response.OK(c, cfg)
	})

	admin.PUT("/notify/events", func(c *gin.Context) {
		var cfg service.NotifyEventsConfig
		if err := c.ShouldBindJSON(&cfg); err != nil {
			response.Error(c, 400, 4001, "请求格式无效")
			return
		}
		saved, err := notifier.SaveEvents(c.Request.Context(), cfg)
		if err != nil {
			response.Error(c, 500, 5000, "保存事件配置失败")
			return
		}
		response.OK(c, saved)
	})
}
