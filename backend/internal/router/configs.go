package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerConfigRoutes(group *gin.RouterGroup, configs *service.SystemConfigService, deps Dependencies) {
	group.GET("/configs", func(c *gin.Context) {
		items, err := configs.List(c.Request.Context())
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "list configs failed")
			return
		}
		payload := authConfig(deps.Config, c.Request)
		applyBrandConfigs(c.Request.Context(), deps.DB, &payload)
		response.OK(c, gin.H{"configs": items, "auth": payload})
	})

	group.PUT("/configs", func(c *gin.Context) {
		var input service.SystemConfigInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4001, "invalid request body")
			return
		}
		items, err := configs.Update(c.Request.Context(), input)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "update configs failed")
			return
		}
		payload := authConfig(deps.Config, c.Request)
		applyBrandConfigs(c.Request.Context(), deps.DB, &payload)
		response.OK(c, gin.H{"configs": items, "auth": payload})
	})
}
