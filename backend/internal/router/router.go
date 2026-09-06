package router

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/cache"
	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/service"
	"gravitylink/backend/internal/web"
)

type Dependencies struct {
	Config      config.Config
	DB          *gorm.DB
	Redis       *redis.Client
	Logger      *slog.Logger
	ResetSystem func(context.Context, uint64) error
}

func New(deps Dependencies) *gin.Engine {
	if deps.Config.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())

	domainCache := service.NewDomainCache(deps.DB)
	if err := domainCache.Load(); err != nil {
		deps.Logger.Warn("load domain cache failed", "error", err)
	}
	routingService := service.NewRoutingService(deps.DB, deps.Redis)
	linkService := service.NewLinkService(deps.DB, cache.NewLinkCache(deps.Redis), routingService)
	domainService := service.NewDomainService(deps.DB, domainCache)
	templates := web.MustLoadTemplates()
	landingService := service.NewLandingService(deps.DB, routingService, templates)
	authService := service.NewAuthService(deps.DB)
	publicPageService := service.NewPublicPageService(deps.DB, templates)
	systemConfigService := service.NewSystemConfigService(deps.DB)
	statService := service.NewStatService(deps.DB, deps.Redis)
	geoResolver := service.NewGeoResolver(deps.Config.GeoDBPath, deps.Logger)
	accessRecorder := service.NewAccessRecorder(deps.Redis, geoResolver)

	api := engine.Group("/api/v1")
	registerHealthRoutes(api, deps)
	registerAuthRoutes(api, deps, authService)

	protectedAPI := api.Group("")
	protectedAPI.Use(middleware.AuthRequired(deps.Config, deps.Logger, deps.DB))
	registerLinkRoutes(protectedAPI, linkService, deps.Config)
	registerLandingRoutes(protectedAPI, landingService, deps.Config)
	registerStatRoutes(protectedAPI, statService)

	adminAPI := engine.Group("/api/admin")
	adminAPI.Use(middleware.AuthRequired(deps.Config, deps.Logger, deps.DB), middleware.RequireRole("admin"))
	registerDomainRoutes(adminAPI, domainService)
	registerConfigRoutes(adminAPI, systemConfigService, deps)
	registerUserRoutes(adminAPI, deps.DB)
	registerSystemRoutes(adminAPI, deps)

	registerAssetRoutes(engine)
	registerPublicRoutes(engine, deps, domainCache, linkService, landingService, publicPageService, accessRecorder)

	return engine
}
