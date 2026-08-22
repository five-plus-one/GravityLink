package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerSystemRoutes(group *gin.RouterGroup, deps Dependencies) {
	system := group.Group("/system")
	system.Use(middleware.RequireSuperAdmin())
	system.POST("/reset", func(c *gin.Context) {
		var input struct {
			Confirmation string `json:"confirmation" binding:"required"`
			Password     string `json:"password"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Confirmation != "RESET GRAVITYLINK" {
			response.Error(c, http.StatusBadRequest, 4000, "confirmation phrase does not match")
			return
		}
		value, exists := c.Get(middleware.ContextUserKey)
		user, ok := value.(middleware.AuthUser)
		if !exists || !ok {
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			return
		}
		authService := service.NewAuthService(deps.DB)
		if user.AuthSource == model.AuthSourceLocal {
			if err := authService.VerifyLocalPassword(user.ID, input.Password); err != nil {
				response.Error(c, http.StatusUnauthorized, 4401, "password verification failed")
				return
			}
		} else if strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")) == "" {
			response.Error(c, http.StatusUnauthorized, 4401, "Logto reauthentication is required")
			return
		}
		if deps.ResetSystem == nil {
			response.Error(c, http.StatusNotImplemented, 5000, "system reset is unavailable")
			return
		}
		if err := deps.ResetSystem(c.Request.Context(), user.ID); err != nil {
			deps.Logger.Error("system reset failed", "error", err)
			response.Error(c, http.StatusInternalServerError, 5000, "system reset failed")
			return
		}
		response.OK(c, map[string]bool{"setup_required": true})
	})
}
