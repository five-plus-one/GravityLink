package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/response"
)

type authConfigPayload struct {
	AuthDisabled          bool     `json:"auth_disabled"`
	Issuer                string   `json:"issuer"`
	ClientID              string   `json:"client_id"`
	Audience              string   `json:"audience"`
	Scopes                string   `json:"scopes"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	LogoutEndpoint        string   `json:"logout_endpoint"`
	RedirectURI           string   `json:"redirect_uri"`
	AllowedRoles          []string `json:"allowed_roles"`
}

func registerAuthRoutes(api *gin.RouterGroup, deps Dependencies) {
	api.GET("/auth/config", func(c *gin.Context) {
		response.OK(c, authConfig(deps.Config, c.Request))
	})

	api.GET("/auth/me", middleware.AuthRequired(deps.Config, deps.Logger), func(c *gin.Context) {
		value, exists := c.Get(middleware.ContextUserKey)
		if !exists {
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			return
		}
		response.OK(c, value)
	})
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
		AuthDisabled:          cfg.AuthDisabled,
		Issuer:                issuer,
		ClientID:              cfg.LogtoAppID,
		Audience:              cfg.LogtoAudience,
		Scopes:                cfg.LogtoScopes,
		AuthorizationEndpoint: issuerEndpoint(issuer, "auth"),
		TokenEndpoint:         issuerEndpoint(issuer, "token"),
		LogoutEndpoint:        issuerEndpoint(issuer, "session/end"),
		RedirectURI:           adminBaseURL + "/auth/callback",
		AllowedRoles:          cfg.AdminAllowedRoles,
	}
}

func issuerEndpoint(issuer string, suffix string) string {
	if issuer == "" {
		return ""
	}
	return issuer + "/" + suffix
}
