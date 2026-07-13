package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/service"
)

const ContextDomainKey = "gravitylink_domain"

func HostRouter(domains *service.DomainCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain, ok := domains.Get(c.Request.Host)
		if !ok {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		c.Set(ContextDomainKey, domain)
		c.Next()
	}
}
