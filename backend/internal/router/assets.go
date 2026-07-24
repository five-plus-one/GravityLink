package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/web"
)

func registerAssetRoutes(engine *gin.Engine) {
	engine.StaticFS("/assets", http.FS(web.Assets))
}
