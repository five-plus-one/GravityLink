package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/config"
)

type Dependencies struct {
	Config config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Logger *slog.Logger
}

func New(deps Dependencies) *gin.Engine {
	if deps.Config.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())

	api := engine.Group("/api/v1")
	registerHealthRoutes(api, deps)

	return engine
}
