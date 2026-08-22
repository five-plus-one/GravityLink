package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

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
}

func registerAuthRoutes(api *gin.RouterGroup, deps Dependencies, authService *service.AuthService) {
	api.GET("/auth/config", func(c *gin.Context) {
		response.OK(c, authConfig(deps.Config, c.Request))
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
	}
}

func issuerEndpoint(issuer string, suffix string) string {
	if issuer == "" {
		return ""
	}
	return issuer + "/" + suffix
}
