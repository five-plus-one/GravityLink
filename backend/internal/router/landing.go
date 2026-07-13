package router

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerLandingRoutes(group *gin.RouterGroup, landings *service.LandingService) {
	group.GET("/landing-pages", func(c *gin.Context) {
		items, err := landings.List(c.Request.Context())
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "list landing pages failed")
			return
		}
		response.OK(c, gin.H{"items": items, "total": len(items)})
	})

	group.POST("/landing-pages", func(c *gin.Context) {
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
}

func writeLandingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLandingNotFound):
		response.Error(c, http.StatusNotFound, 4004, "landing page not found")
	case errors.Is(err, service.ErrInvalidTemplate):
		response.Error(c, http.StatusBadRequest, 4001, "invalid template")
	default:
		response.Error(c, http.StatusInternalServerError, 5000, "internal server error")
	}
}
