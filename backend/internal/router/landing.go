package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerLandingRoutes(group *gin.RouterGroup, landings *service.LandingService, cfg config.Config) {
	group.GET("/landing-pages/:id/preview", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		html, err := landings.Preview(c.Request.Context(), id)
		if err != nil {
			writeLandingError(c, err)
			return
		}
		html = strings.ReplaceAll(html, `<script src="/assets/landing/gravitylink-landing.js"></script>`, "")
		response.OK(c, gin.H{"html": html})
	})
	group.PUT("/landing-pages/:id", middleware.RequireAnyRole(cfg.AdminAllowedRoles...), func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		var input service.LandingInput
		if c.ShouldBindJSON(&input) != nil {
			response.Error(c, 400, 4001, "配置格式无效")
			return
		}
		page, err := landings.Update(c.Request.Context(), id, input)
		if err != nil {
			writeLandingError(c, err)
			return
		}
		response.OK(c, page)
	})
	group.GET("/landing-pages", func(c *gin.Context) {
		items, err := landings.List(c.Request.Context())
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "list landing pages failed")
			return
		}
		response.OK(c, gin.H{"items": items, "total": len(items)})
	})

	group.POST("/landing-pages", middleware.RequireAnyRole(cfg.AdminAllowedRoles...), func(c *gin.Context) {
		var input service.LandingInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4001, "invalid request body")
			return
		}
		page, err := landings.Create(c.Request.Context(), input)
		if err != nil {
			writeLandingError(c, err)
			return
		}
		response.OK(c, page)
	})

	group.DELETE("/landing-pages/:id", middleware.RequireAnyRole(cfg.AdminAllowedRoles...), func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		if err := landings.Delete(c.Request.Context(), id); err != nil {
			writeLandingError(c, err)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})

	// P1：草稿预览——不落库直接渲染表单当前内容，支持编辑中实时预览。
	group.POST("/landing-pages/preview-draft", middleware.RequireAnyRole(cfg.AdminAllowedRoles...), func(c *gin.Context) {
		var input service.LandingInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4001, "invalid request body")
			return
		}
		html, err := landings.PreviewDraft(input)
		if err != nil {
			writeLandingError(c, err)
			return
		}
		html = strings.ReplaceAll(html, `<script src="/assets/landing/gravitylink-landing.js"></script>`, "")
		response.OK(c, gin.H{"html": html})
	})
}

func writeLandingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLandingNotFound):
		response.Error(c, http.StatusNotFound, 4004, "landing page not found")
	case errors.Is(err, service.ErrInvalidTemplate):
		response.Error(c, http.StatusBadRequest, 4001, "invalid template")
	case errors.Is(err, service.ErrLandingInUse):
		response.Error(c, http.StatusConflict, 4009, "落地页仍被链接引用，请先在链接管理中解绑或删除相关链接")
	default:
		response.Error(c, http.StatusInternalServerError, 5000, "internal server error")
	}
}
