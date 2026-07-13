package router

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerPublicRoutes(engine *gin.Engine, deps Dependencies, domains *service.DomainCache, links *service.LinkService, landings *service.LandingService, recorder *service.AccessRecorder) {
	public := engine.Group("/")
	public.Use(middleware.HostRouter(domains))
	public.GET("/:code", dispatchByDomainType(deps, links, landings, recorder))
}

func dispatchByDomainType(deps Dependencies, links *service.LinkService, landings *service.LandingService, recorder *service.AccessRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(middleware.ContextDomainKey)
		if !exists {
			response.Error(c, http.StatusNotFound, 4101, "domain not found")
			return
		}

		domain, ok := value.(model.Domain)
		if !ok {
			response.Error(c, http.StatusInternalServerError, 5000, "domain context invalid")
			return
		}

		code := c.Param("code")
		switch domain.Type {
		case model.DomainTypeEntry:
			result, err := links.Resolve(c.Request.Context(), code)
			if err != nil {
				writeResolveError(c, err)
				return
			}
			recorder.RecordAsync(service.AccessEvent{
				LinkID:     result.Link.ID,
				IP:         c.ClientIP(),
				UserAgent:  c.Request.UserAgent(),
				Referer:    c.Request.Referer(),
				VisitedAt:  deps.Config.Now(),
				ViaTransit: false,
			})
			c.Redirect(result.Status, result.TargetURL)
		case model.DomainTypeTransit:
			response.OK(c, gin.H{"handler": "transit", "host": domain.Host, "code": code})
		case model.DomainTypeLanding:
			html, status, err := landings.RenderByCode(c.Request.Context(), code)
			if err != nil {
				writeResolveError(c, err)
				return
			}
			c.Data(status, "text/html; charset=utf-8", []byte(html))
		default:
			deps.Logger.Warn("unsupported domain type", "host", domain.Host, "type", domain.Type)
			response.Error(c, http.StatusNotFound, 4101, "domain type unsupported")
		}
	}
}

func writeResolveError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLinkNotFound), errors.Is(err, service.ErrLinkDisabled):
		c.AbortWithStatus(http.StatusNotFound)
	case errors.Is(err, service.ErrLinkExpired):
		c.AbortWithStatus(http.StatusGone)
	case errors.Is(err, service.ErrUnsupportedLink), errors.Is(err, service.ErrTargetUnavailable):
		c.AbortWithStatus(http.StatusNotFound)
	default:
		response.Error(c, http.StatusInternalServerError, 5000, "resolve link failed")
	}
}
