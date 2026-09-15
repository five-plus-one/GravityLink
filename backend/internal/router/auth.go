package router

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

type authConfigPayload struct {
	Issuer                string   `json:"issuer"`
	ClientID              string   `json:"client_id"`
	Audience              string   `json:"audience"`
	Scopes                string   `json:"scopes"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	LogoutEndpoint        string   `json:"logout_endpoint"`
	RedirectURI           string   `json:"redirect_uri"`
	AllowedRoles          []string `json:"allowed_roles"`
	Mode                  string   `json:"mode"`
	LocalEnabled          bool     `json:"local_enabled"`
	// 品牌展示（登录页/管理端），来自 system_configs，未配置时用默认值
	BrandName string `json:"brand_name"`
	BrandLogo string `json:"brand_logo"`
	AuthLabel string `json:"auth_label"`
	AuthLogo  string `json:"auth_logo"`
}

// brandDefaults 未配置时的展示文案。
const (
	defaultBrandName = "GravityLink"
	defaultAuthLabel = "Logto"
)

func registerAuthRoutes(api *gin.RouterGroup, deps Dependencies, authService *service.AuthService) {
	api.GET("/auth/config", func(c *gin.Context) {
		payload := authConfig(deps.Config, c.Request)
		applyBrandConfigs(c.Request.Context(), deps.DB, &payload)
		response.OK(c, payload)
	})

	api.POST("/auth/local/login", func(c *gin.Context) {
		var input struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4000, "username and password are required")
			return
		}
		user, token, err := authService.LoginLocal(input.Username, input.Password)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, 4401, "invalid username or password")
			return
		}
		secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(middleware.LocalSessionCookie, token, int((12 * time.Hour).Seconds()), "/", "", secure, true)
		response.OK(c, authUserPayload(user))
	})

	api.POST("/auth/logout", func(c *gin.Context) {
		if token, err := c.Cookie(middleware.LocalSessionCookie); err == nil {
			_ = authService.RevokeSession(token)
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(middleware.LocalSessionCookie, "", -1, "/", "", false, true)
		response.OK(c, map[string]bool{"logged_out": true})
	})

	api.GET("/auth/me", middleware.AuthRequired(deps.Config, deps.Logger, deps.DB), func(c *gin.Context) {
		value, exists := c.Get(middleware.ContextUserKey)
		if !exists {
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			return
		}
		response.OK(c, value)
	})
}

func authUserPayload(user model.User) map[string]any {
	email := ""
	subject := ""
	if user.Email != nil {
		email = *user.Email
	}
	if user.SSOID != nil {
		subject = *user.SSOID
	}
	return map[string]any{
		"id": user.ID, "subject": subject, "email": email, "username": user.Username,
		"role": user.Role, "status": user.Status, "auth_source": user.AuthSource,
	}
}

func authConfig(cfg config.Config, req *http.Request) authConfigPayload {
	issuer := strings.TrimRight(cfg.LogtoIssuer, "/")
	adminBaseURL := strings.TrimRight(cfg.AdminBaseURL, "/")
	if adminBaseURL == "" && req != nil {
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		if forwarded := req.Header.Get("X-Forwarded-Proto"); forwarded != "" {
			scheme = forwarded
		}
		adminBaseURL = scheme + "://" + req.Host
	}

	return authConfigPayload{
		Issuer:                issuer,
		ClientID:              cfg.LogtoAppID,
		Audience:              cfg.LogtoAudience,
		Scopes:                cfg.LogtoScopes,
		AuthorizationEndpoint: issuerEndpoint(issuer, "auth"),
		TokenEndpoint:         issuerEndpoint(issuer, "token"),
		LogoutEndpoint:        issuerEndpoint(issuer, "session/end"),
		RedirectURI:           adminBaseURL + "/auth/callback",
		AllowedRoles:          cfg.AdminAllowedRoles,
		Mode:                  cfg.AuthMode,
		LocalEnabled:          cfg.AuthMode == model.AuthSourceLocal,
		BrandName:             defaultBrandName,
		AuthLabel:             defaultAuthLabel,
	}
}

// applyBrandConfigs 从 system_configs 读取品牌展示配置（登录页无需鉴权即可拿到）。
func applyBrandConfigs(ctx context.Context, db *gorm.DB, payload *authConfigPayload) {
	if db == nil {
		return
	}
	var rows []struct {
		KeyName string
		Value   string
	}
	if err := db.WithContext(ctx).Table("system_configs").
		Where("key_name IN ?", []string{"brand.name", "brand.logo_url", "brand.auth_label", "brand.auth_logo_url"}).
		Scan(&rows).Error; err != nil {
		return
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.KeyName] = strings.TrimSpace(row.Value)
	}
	if v := values["brand.name"]; v != "" {
		payload.BrandName = v
	}
	payload.BrandLogo = values["brand.logo_url"]
	if v := values["brand.auth_label"]; v != "" {
		payload.AuthLabel = v
	}
	payload.AuthLogo = values["brand.auth_logo_url"]
}

func issuerEndpoint(issuer string, suffix string) string {
	if issuer == "" {
		return ""
	}
	return issuer + "/" + suffix
}
