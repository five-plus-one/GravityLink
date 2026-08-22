package middleware

import (
	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/service"
)

const ContextDomainKey = "gravitylink_domain"

// HostRouter 按 Host 解析域名。未注册的 Host 返回品牌化 404 页，
// 而不是无 body 的浏览器原生错误页。
func HostRouter(domains *service.DomainCache, cfg config.Config, pages *service.PublicPageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain, ok := domains.Get(c.Request.Host)
		if !ok {
			html, status := pages.NotFound(c.Request.Context())
			c.Data(status, "text/html; charset=utf-8", []byte(html))
			c.Abort()
			return
		}

		c.Set(ContextDomainKey, domain)
		c.Next()
	}
}
