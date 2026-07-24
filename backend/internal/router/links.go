package router

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerLinkRoutes(group *gin.RouterGroup, links *service.LinkService, cfg config.Config) {
	group.GET("/links", func(c *gin.Context) {
		items, err := links.List(c.Request.Context(), c.Query("type"))
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "list links failed")
			return
		}
		response.OK(c, gin.H{"items": items, "total": len(items)})
	})

	adminOnly := middleware.RequireAnyRole(cfg.AdminAllowedRoles...)

	group.POST("/links", adminOnly, func(c *gin.Context) {
		var input service.CreateLinkInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4001, "invalid request body")
			return
		}

		link, err := links.Create(c.Request.Context(), input)
		writeLinkResult(c, link, err)
	})

	group.GET("/links/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		link, err := links.Get(c.Request.Context(), id)
		writeLinkResult(c, link, err)
	})

	group.PUT("/links/:id", adminOnly, func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}

		var input service.UpdateLinkInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4001, "invalid request body")
			return
		}

		link, err := links.Update(c.Request.Context(), id, input)
		writeLinkResult(c, link, err)
	})

	group.DELETE("/links/:id", adminOnly, func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		if err := links.Delete(c.Request.Context(), id); err != nil {
			writeServiceError(c, err)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
}

func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4001, "invalid id")
		return 0, false
	}
	return id, true
}

func writeLinkResult(c *gin.Context, data interface{}, err error) {
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, data)
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLinkNotFound):
		response.Error(c, http.StatusNotFound, 4004, "link not found")
	case errors.Is(err, service.ErrCodeConflict):
		response.Error(c, http.StatusConflict, 4001, "code already exists")
	case errors.Is(err, service.ErrInvalidCode):
		response.Error(c, http.StatusBadRequest, 4001, "invalid code")
	case errors.Is(err, service.ErrInvalidTargetURL):
		response.Error(c, http.StatusBadRequest, 4002, "invalid target url")
	case errors.Is(err, service.ErrUnsupportedLink):
		response.Error(c, http.StatusBadRequest, 4003, "unsupported link type")
	default:
		response.Error(c, http.StatusInternalServerError, 5000, "internal server error")
	}
}
