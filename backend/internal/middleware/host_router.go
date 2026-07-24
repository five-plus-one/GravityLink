package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/service"
)

const ContextDomainKey = "gravitylink_domain"

func HostRouter(domains *service.DomainCache, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain, ok := domains.Get(c.Request.Host)
		if !ok {
			if cfg.AuthDisabled || cfg.AppEnv == "development" {
				c.Set(ContextDomainKey, model.Domain{Host: c.Request.Host, Type: model.DomainTypeEntry, Scheme: "http"})
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		c.Set(ContextDomainKey, domain)
		c.Next()
	}
}
