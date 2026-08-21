package middleware

import (
	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/service"
)

const ContextDomainKey = "gravitylink_domain"

// HostRouter 按 Host 解析域名。未注册的 Host 在生产环境返回品牌化 404 页，
// 而不是无 body 的浏览器原生错误页。开发环境/认证关闭时回退为入口域以便本地调试。
func HostRouter(domains *service.DomainCache, cfg config.Config, pages *service.PublicPageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain, ok := domains.Get(c.Request.Host)
		if !ok {
			if cfg.AuthDisabled || cfg.AppEnv == "development" {
				c.Set(ContextDomainKey, model.Domain{Host: c.Request.Host, Type: model.DomainTypeEntry, Scheme: "http"})
				c.Next()
				return
			}
			html, status := pages.NotFound(c.Request.Context())
			c.Data(status, "text/html; charset=utf-8", []byte(html))
			c.Abort()
			return
		}

		c.Set(ContextDomainKey, domain)
		c.Next()
	}
}
