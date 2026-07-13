package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/response"
)

func registerHealthRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		sqlDB, err := deps.DB.DB()
		if err != nil {
			response.Error(c, 500, 5000, "database handle unavailable")
			return
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			response.Error(c, 500, 5000, "mysql unavailable")
			return
		}
		if err := deps.Redis.Ping(ctx).Err(); err != nil {
			response.Error(c, 500, 5000, "redis unavailable")
			return
		}

		response.OK(c, gin.H{
			"service": "gravitylink",
			"status":  "ok",
		})
	})
}
