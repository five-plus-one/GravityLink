package middleware

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/model"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"正常 Bearer", "Bearer abc.def.ghi", "abc.def.ghi"},
		{"前缀大小写不匹配拒绝", "bearer abc", ""},
		{"无 Bearer 前缀", "Basic abc", ""},
		{"仅前缀无 token", "Bearer ", ""},
		{"空 header", "", ""},
		{"token 带首尾空格被清理", "Bearer  token  ", "token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bearerToken(tt.header); got != tt.want {
				t.Errorf("bearerToken(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestValidatePayload(t *testing.T) {
	verifier := JWTVerifier{config: config.Config{
		LogtoIssuer:   "https://issuer.example.com",
		LogtoAudience: "https://api.example.com",
	}}
	now := time.Now().Unix()

	tests := []struct {
		name    string
		payload jwtPayload
		wantErr bool
	}{
		{"合法 payload", jwtPayload{"iss": "https://issuer.example.com", "aud": "https://api.example.com", "exp": float64(now + 3600)}, false},
		{"issuer 不匹配", jwtPayload{"iss": "https://evil.com", "aud": "https://api.example.com", "exp": float64(now + 3600)}, true},
		{"audience 不匹配", jwtPayload{"iss": "https://issuer.example.com", "aud": "https://other.com", "exp": float64(now + 3600)}, true},
		{"audience 为数组且包含", jwtPayload{"iss": "https://issuer.example.com", "aud": []interface{}{"https://other.com", "https://api.example.com"}, "exp": float64(now + 3600)}, false},
		{"token 已过期", jwtPayload{"iss": "https://issuer.example.com", "aud": "https://api.example.com", "exp": float64(now - 1)}, true},
		{"缺少 exp 视为不过期", jwtPayload{"iss": "https://issuer.example.com", "aud": "https://api.example.com"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifier.validatePayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRSAPEublicKeyFromJWK(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := jwkKey{
		KeyID: "k1", KeyType: "RSA", Algorithm: "RS256", Use: "sig",
		Modulus:  base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		Exponent: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
	parsed, err := rsaPublicKey(jwk)
	if err != nil {
		t.Fatalf("rsaPublicKey() error = %v", err)
	}
	if parsed.N.Cmp(key.N) != 0 || parsed.E != key.E {
		t.Error("parsed public key mismatch")
	}

	t.Run("空 exponent 拒绝", func(t *testing.T) {
		if _, err := rsaPublicKey(jwkKey{Modulus: jwk.Modulus}); err == nil {
			t.Error("expected error for empty exponent")
		}
	})
}

// TestJWTVerifierVerify 用真实 RSA 密钥对 + httptest JWKS 服务走完整验签流程。
func TestJWTVerifierVerify(t *testing.T) {
	gin.SetMode(gin.TestMode)

	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	jwks := jwksResponse{Keys: []jwkKey{{
		KeyID: "test-kid", KeyType: "RSA", Algorithm: "RS256", Use: "sig",
		Modulus:  base64.RawURLEncoding.EncodeToString(signingKey.N.Bytes()),
		Exponent: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(signingKey.E)).Bytes()),
	}}}
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	verifier := JWTVerifier{config: config.Config{
		LogtoJWKSURL:  jwksServer.URL,
		LogtoIssuer:   "https://issuer.example.com",
		LogtoAudience: "https://api.example.com",
	}}

	encodePart := func(v interface{}) string {
		data, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(data)
	}
	signToken := func(key *rsa.PrivateKey, header, payload string) string {
		digest := sha256.Sum256([]byte(header + "." + payload))
		sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if err != nil {
			t.Fatal(err)
		}
		return header + "." + payload + "." + base64.RawURLEncoding.EncodeToString(sig)
	}
	makeToken := func(key *rsa.PrivateKey, payload jwtPayload, alg string, kid string) string {
		if alg == "" {
			alg = "RS256"
		}
		return signToken(key, encodePart(jwtHeader{Algorithm: alg, KeyID: kid}), encodePart(payload))
	}

	validPayload := func() jwtPayload {
		return jwtPayload{
			"iss": "https://issuer.example.com",
			"aud": "https://api.example.com",
			"exp": float64(time.Now().Add(time.Hour).Unix()),
			"sub": "user-123",
		}
	}

	t.Run("合法 token 通过并提取身份", func(t *testing.T) {
		user, err := verifier.Verify(t.Context(), makeToken(signingKey, validPayload(), "", "test-kid"))
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}
		if user.Subject != "user-123" || user.AuthSource != model.AuthSourceLogto {
			t.Errorf("unexpected user: %+v", user)
		}
	})

	t.Run("签名不符拒绝", func(t *testing.T) {
		if _, err := verifier.Verify(t.Context(), makeToken(otherKey, validPayload(), "", "test-kid")); err == nil {
			t.Error("expected error for wrong signature")
		}
	})

	t.Run("过期 token 拒绝", func(t *testing.T) {
		payload := validPayload()
		payload["exp"] = float64(time.Now().Add(-time.Hour).Unix())
		if _, err := verifier.Verify(t.Context(), makeToken(signingKey, payload, "", "test-kid")); err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("issuer 不符拒绝", func(t *testing.T) {
		payload := validPayload()
		payload["iss"] = "https://evil.com"
		if _, err := verifier.Verify(t.Context(), makeToken(signingKey, payload, "", "test-kid")); err == nil {
			t.Error("expected error for wrong issuer")
		}
	})

	t.Run("非 RS256 算法拒绝", func(t *testing.T) {
		if _, err := verifier.Verify(t.Context(), makeToken(signingKey, validPayload(), "HS256", "test-kid")); err == nil {
			t.Error("expected error for non-RS256 alg")
		}
	})

	t.Run("未知 kid 拒绝", func(t *testing.T) {
		if _, err := verifier.Verify(t.Context(), makeToken(signingKey, validPayload(), "", "unknown-kid")); err == nil {
			t.Error("expected error for unknown kid")
		}
	})

	t.Run("格式非法 token 拒绝", func(t *testing.T) {
		if _, err := verifier.Verify(t.Context(), "not-a-jwt"); err == nil {
			t.Error("expected error for malformed token")
		}
	})
}

// TestRequireAnyRole RBAC 中间件行为。
func TestRequireAnyRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	buildRouter := func(setUser func(c *gin.Context), roles ...string) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if setUser != nil {
				setUser(c)
			}
			c.Next()
		})
		r.GET("/protected", RequireAnyRole(roles...), func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})
		return r
	}
	do := func(r *gin.Engine) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		r.ServeHTTP(w, req)
		return w.Code
	}
	setUser := func(role string) func(c *gin.Context) {
		return func(c *gin.Context) {
			c.Set(ContextUserKey, AuthUser{Role: role})
		}
	}

	tests := []struct {
		name    string
		setUser func(c *gin.Context)
		roles   []string
		want    int
	}{
		{"admin 访问 admin 接口", setUser(model.UserRoleAdmin), []string{model.UserRoleAdmin}, http.StatusOK},
		{"user 访问 admin 接口被拒", setUser(model.UserRoleUser), []string{model.UserRoleAdmin}, http.StatusForbidden},
		{"super_admin 无视角色要求", setUser(model.UserRoleSuperAdmin), []string{model.UserRoleAdmin}, http.StatusOK},
		{"未设置用户 401", nil, []string{model.UserRoleAdmin}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := do(buildRouter(tt.setUser, tt.roles...)); got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}

	t.Run("RequireSuperAdmin 拒绝普通 admin", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set(ContextUserKey, AuthUser{Role: model.UserRoleAdmin}); c.Next() })
		r.GET("/only-super", RequireSuperAdmin(), func(c *gin.Context) { c.String(http.StatusOK, "ok") })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/only-super", nil))
		if got := w.Code; got != http.StatusForbidden {
			t.Errorf("status = %d, want %d", got, http.StatusForbidden)
		}
	})
}

func TestAuthUserFromModel(t *testing.T) {
	ssoID := "sub-abc"
	email := "a@b.c"
	user := authUserFromModel(model.User{
		ID: 7, Username: "alice", Role: model.UserRoleAdmin, Status: model.StatusActive,
		AuthSource: model.AuthSourceLocal, SSOID: &ssoID, Email: &email,
	})
	if user.ID != 7 || user.Subject != "sub-abc" || user.Email != "a@b.c" || user.Role != model.UserRoleAdmin {
		t.Errorf("unexpected AuthUser: %+v", user)
	}
	if !strings.EqualFold(user.Username, "alice") {
		t.Errorf("unexpected username: %s", user.Username)
	}
}
