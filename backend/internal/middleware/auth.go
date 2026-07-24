package middleware

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

const ContextUserKey = "gravitylink_user"
const LocalSessionCookie = "gravitylink_session"

var errTokenInvalid = errors.New("token invalid")

type AuthUser struct {
	ID         uint64 `json:"id"`
	Subject    string `json:"subject"`
	Email      string `json:"email"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	AuthSource string `json:"auth_source"`
}

func AuthRequired(cfg config.Config, logger *slog.Logger, databases ...*gorm.DB) gin.HandlerFunc {
	verifier := JWTVerifier{config: cfg}
	var db *gorm.DB
	if len(databases) > 0 {
		db = databases[0]
	}

	return func(c *gin.Context) {
		if cfg.AuthDisabled {
			c.Set(ContextUserKey, AuthUser{Subject: "dev", Username: "dev", Role: model.UserRoleSuperAdmin, Status: model.StatusActive, AuthSource: "development"})
			c.Next()
			return
		}

		if db != nil {
			if sessionToken, err := c.Cookie(LocalSessionCookie); err == nil && sessionToken != "" {
				user, err := service.NewAuthService(db).ResolveSession(sessionToken)
				if err == nil {
					c.Set(ContextUserKey, authUserFromModel(user))
					c.Next()
					return
				}
			}
		}

		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			c.Abort()
			return
		}

		identity, err := verifier.Verify(c.Request.Context(), token)
		if err != nil {
			logger.Warn("jwt verify failed", "error", err)
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			c.Abort()
			return
		}
		if db == nil {
			c.Set(ContextUserKey, identity)
			c.Next()
			return
		}
		dbUser, err := service.NewAuthService(db).ProvisionLogto(service.Identity{
			Subject: identity.Subject, Username: identity.Username, Email: identity.Email,
		})
		if err != nil {
			logger.Warn("provision Logto user failed", "error", err)
			response.Error(c, http.StatusInternalServerError, 5000, "account lookup failed")
			c.Abort()
			return
		}
		if dbUser.Status != model.StatusActive {
			response.Error(c, http.StatusForbidden, 4403, "account pending approval")
			c.Abort()
			return
		}
		c.Set(ContextUserKey, authUserFromModel(dbUser))
		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return RequireAnyRole(role)
}

func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(ContextUserKey)
		if !exists {
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			c.Abort()
			return
		}

		user, ok := value.(AuthUser)
		if !ok || !roleAllowed(user.Role, roles) {
			response.Error(c, http.StatusForbidden, 4403, "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

func roleAllowed(userRole string, roles []string) bool {
	if userRole == model.UserRoleSuperAdmin {
		return true
	}
	for _, role := range roles {
		if userRole == role {
			return true
		}
	}
	return false
}

func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(ContextUserKey)
		user, ok := value.(AuthUser)
		if !exists || !ok || user.Role != model.UserRoleSuperAdmin {
			response.Error(c, http.StatusForbidden, 4403, "super administrator required")
			c.Abort()
			return
		}
		c.Next()
	}
}

type JWTVerifier struct {
	config config.Config
}

func VerifyToken(ctx context.Context, cfg config.Config, token string) (AuthUser, error) {
	return JWTVerifier{config: cfg}.Verify(ctx, token)
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

type jwtPayload map[string]interface{}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	KeyID     string `json:"kid"`
	KeyType   string `json:"kty"`
	Algorithm string `json:"alg"`
	Use       string `json:"use"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}

func (v JWTVerifier) Verify(ctx context.Context, token string) (AuthUser, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return AuthUser{}, errTokenInvalid
	}

	var header jwtHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return AuthUser{}, err
	}
	if header.Algorithm != "RS256" || header.KeyID == "" {
		return AuthUser{}, errTokenInvalid
	}

	var payload jwtPayload
	if err := decodeJWTPart(parts[1], &payload); err != nil {
		return AuthUser{}, err
	}
	if err := v.validatePayload(payload); err != nil {
		return AuthUser{}, err
	}

	key, err := v.fetchKey(ctx, header.KeyID)
	if err != nil {
		return AuthUser{}, err
	}

	signed := []byte(parts[0] + "." + parts[1])
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return AuthUser{}, err
	}
	digest := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return AuthUser{}, errTokenInvalid
	}

	return userFromPayload(payload), nil
}

func (v JWTVerifier) validatePayload(payload jwtPayload) error {
	if v.config.LogtoIssuer != "" && stringClaim(payload, "iss") != v.config.LogtoIssuer {
		return errTokenInvalid
	}
	if v.config.LogtoAudience != "" && !audienceContains(payload["aud"], v.config.LogtoAudience) {
		return errTokenInvalid
	}
	if exp, ok := numberClaim(payload, "exp"); ok && time.Now().Unix() >= int64(exp) {
		return errTokenInvalid
	}
	return nil
}

func (v JWTVerifier) fetchKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	jwksURL := v.config.LogtoJWKSURL
	if jwksURL == "" && v.config.LogtoIssuer != "" {
		jwksURL = strings.TrimRight(v.config.LogtoIssuer, "/") + "/oidc/jwks"
	}
	if jwksURL == "" {
		return nil, errTokenInvalid
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errTokenInvalid
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}
	for _, key := range jwks.Keys {
		if key.KeyID == kid && key.KeyType == "RSA" {
			return rsaPublicKey(key)
		}
	}
	return nil, errTokenInvalid
}

func rsaPublicKey(key jwkKey) (*rsa.PublicKey, error) {
	modulus, err := base64.RawURLEncoding.DecodeString(key.Modulus)
	if err != nil {
		return nil, err
	}
	exponent, err := base64.RawURLEncoding.DecodeString(key.Exponent)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(modulus)
	e := new(big.Int).SetBytes(exponent).Int64()
	if e == 0 {
		return nil, errTokenInvalid
	}

	return &rsa.PublicKey{N: n, E: int(e)}, nil
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func decodeJWTPart(part string, target interface{}) error {
	data, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func userFromPayload(payload jwtPayload) AuthUser {
	return AuthUser{
		Subject:    stringClaim(payload, "sub"),
		Email:      stringClaim(payload, "email"),
		Username:   firstNonEmpty(stringClaim(payload, "username"), stringClaim(payload, "name")),
		Status:     model.StatusActive,
		AuthSource: model.AuthSourceLogto,
	}
}

func authUserFromModel(user model.User) AuthUser {
	result := AuthUser{
		ID: user.ID, Username: user.Username, Role: user.Role, Status: user.Status, AuthSource: user.AuthSource,
	}
	if user.SSOID != nil {
		result.Subject = *user.SSOID
	}
	if user.Email != nil {
		result.Email = *user.Email
	}
	return result
}

func stringClaim(payload jwtPayload, key string) string {
	value, ok := payload[key].(string)
	if !ok {
		return ""
	}
	return value
}

func numberClaim(payload jwtPayload, key string) (float64, bool) {
	value, ok := payload[key].(float64)
	return value, ok
}

func audienceContains(value interface{}, expected string) bool {
	switch audience := value.(type) {
	case string:
		return audience == expected
	case []interface{}:
		for _, item := range audience {
			if item == expected {
				return true
			}
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
