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

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/response"
)

const ContextUserKey = "gravitylink_user"

var errTokenInvalid = errors.New("token invalid")

type AuthUser struct {
	Subject  string
	Email    string
	Username string
	Role     string
}

func AuthRequired(cfg config.Config, logger *slog.Logger) gin.HandlerFunc {
	verifier := JWTVerifier{config: cfg}

	return func(c *gin.Context) {
		if cfg.AuthDisabled {
			c.Set(ContextUserKey, AuthUser{Subject: "dev", Username: "dev", Role: "admin"})
			c.Next()
			return
		}

		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			c.Abort()
			return
		}

		user, err := verifier.Verify(c.Request.Context(), token)
		if err != nil {
			logger.Warn("jwt verify failed", "error", err)
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			c.Abort()
			return
		}

		c.Set(ContextUserKey, user)
		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(ContextUserKey)
		if !exists {
			response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
			c.Abort()
			return
		}

		user, ok := value.(AuthUser)
		if !ok || (user.Role != role && user.Role != "admin") {
			response.Error(c, http.StatusForbidden, 4403, "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

type JWTVerifier struct {
	config config.Config
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
		Subject:  stringClaim(payload, "sub"),
		Email:    stringClaim(payload, "email"),
		Username: firstNonEmpty(stringClaim(payload, "username"), stringClaim(payload, "name")),
		Role:     roleFromPayload(payload),
	}
}

func roleFromPayload(payload jwtPayload) string {
	if role := stringClaim(payload, "role"); role != "" {
		return role
	}
	if roles, ok := payload["roles"].([]interface{}); ok {
		for _, role := range roles {
			if value, ok := role.(string); ok && value == "admin" {
				return "admin"
			}
		}
	}
	return "user"
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
