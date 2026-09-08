package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

var (
	ErrAPIKeyNotFound      = errors.New("api key not found")
	ErrAPIKeyDisabled      = errors.New("api key disabled")
	ErrAPIKeyExpired       = errors.New("api key expired")
	ErrAPIKeyIPDenied      = errors.New("ip not in whitelist")
	ErrAPIKeyQuotaExceeded = errors.New("quota exceeded")
	ErrAPIKeySignInvalid   = errors.New("signature invalid")
)

type APIKeyService struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewAPIKeyService(db *gorm.DB, redis *redis.Client) *APIKeyService {
	return &APIKeyService{db: db, redis: redis}
}

type CreateKeyInput struct {
	Name        string     `json:"name"`
	SignEnabled bool       `json:"sign_enabled"`
	Quota       *uint      `json:"quota"`
	ExpireAt    *time.Time `json:"expire_at"`
	IPWhitelist string     `json:"ip_whitelist"`
}

type CreateKeyResult struct {
	*model.ApiKey
	Token      string `json:"token"`
	HmacSecret string `json:"hmac_secret,omitempty"`
}

func (s *APIKeyService) CreateKey(ctx context.Context, input CreateKeyInput, createdBy uint64) (*CreateKeyResult, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("key name is required")
	}
	token, err := generateToken("gl_live_", 24)
	if err != nil {
		return nil, err
	}
	tokenHash := SHA256Hex(token)

	key := &model.ApiKey{
		Name:        name,
		TokenHash:   tokenHash,
		SignEnabled: input.SignEnabled,
		Quota:       input.Quota,
		ExpireAt:    input.ExpireAt,
		Status:      model.StatusActive,
		CreatedBy:   createdBy,
	}
	if wl := strings.TrimSpace(input.IPWhitelist); wl != "" {
		key.IPWhitelist = &wl
	}

	var hmacSecret string
	if input.SignEnabled {
		hmacSecret, err = generateToken("gl_sec_", 32)
		if err != nil {
			return nil, err
		}
		secretHash := SHA256Hex(hmacSecret)
		key.HmacSecret = &secretHash
	}

	if err := s.db.WithContext(ctx).Create(key).Error; err != nil {
		return nil, err
	}
	return &CreateKeyResult{ApiKey: key, Token: token, HmacSecret: hmacSecret}, nil
}

func (s *APIKeyService) ListKeys(ctx context.Context, createdBy uint64) ([]model.ApiKey, error) {
	var keys []model.ApiKey
	return keys, s.db.WithContext(ctx).Where("created_by = ?", createdBy).Order("id DESC").Find(&keys).Error
}

type UpdateKeyInput struct {
	Name        *string    `json:"name"`
	Status      *string    `json:"status"`
	Quota       *uint      `json:"quota"`
	ExpireAt    *time.Time `json:"expire_at"`
	IPWhitelist *string    `json:"ip_whitelist"`
}

func (s *APIKeyService) UpdateKey(ctx context.Context, id uint64, input UpdateKeyInput) (*model.ApiKey, error) {
	var key model.ApiKey
	if err := s.db.WithContext(ctx).First(&key, id).Error; err != nil {
		return nil, ErrAPIKeyNotFound
	}
	updates := map[string]any{}
	if input.Name != nil {
		updates["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Status != nil && (*input.Status == "active" || *input.Status == "disabled") {
		updates["status"] = *input.Status
	}
	if input.Quota != nil {
		updates["quota"] = input.Quota
	}
	if input.ExpireAt != nil {
		updates["expire_at"] = input.ExpireAt
	}
	if input.IPWhitelist != nil {
		wl := strings.TrimSpace(*input.IPWhitelist)
		if wl == "" {
			updates["ip_whitelist"] = nil
		} else {
			updates["ip_whitelist"] = wl
		}
	}
	if err := s.db.WithContext(ctx).Model(&key).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &key, s.db.WithContext(ctx).First(&key, id).Error
}

func (s *APIKeyService) DeleteKey(ctx context.Context, id, createdBy uint64) error {
	result := s.db.WithContext(ctx).Where("id = ? AND created_by = ?", id, createdBy).Delete(&model.ApiKey{})
	if result.RowsAffected == 0 {
		return ErrAPIKeyNotFound
	}
	return result.Error
}

// ValidateKey 从请求 Bearer token 验证 Key 合法性；signKey 模式下额外校验 HMAC 时间戳签名。
func (s *APIKeyService) ValidateKey(ctx context.Context, bearerToken, ip, timestamp, body, signature string) (*model.ApiKey, error) {
	if bearerToken == "" {
		return nil, ErrAPIKeyNotFound
	}
	tokenHash := SHA256Hex(bearerToken)
	var key model.ApiKey
	err := s.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	if key.Status != model.StatusActive {
		return nil, ErrAPIKeyDisabled
	}
	if key.ExpireAt != nil && time.Now().After(*key.ExpireAt) {
		return nil, ErrAPIKeyExpired
	}
	if key.IPWhitelist != nil && *key.IPWhitelist != "" {
		if !ipInWhitelist(ip, *key.IPWhitelist) {
			return nil, ErrAPIKeyIPDenied
		}
	}
	// 软配额检查（只读；并发下可能微量超发，对业务无影响）
	if key.Quota != nil {
		used, _ := s.redis.Get(ctx, "apikey:used:"+fmt.Sprint(key.ID)).Uint64()
		if used >= uint64(*key.Quota) {
			return nil, ErrAPIKeyQuotaExceeded
		}
	}
	// HMAC 签名校验（Key 级开关开启时才要求）
	if key.SignEnabled && key.HmacSecret != nil {
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil || time.Now().Unix()-ts > 300 || ts-time.Now().Unix() > 300 {
			return nil, ErrAPIKeySignInvalid
		}
		expected := HMACSHA256Hex(*key.HmacSecret, timestamp+body)
		if !hmac.Equal([]byte(expected), []byte(signature)) {
			return nil, ErrAPIKeySignInvalid
		}
	}
	return &key, nil
}

// IncrKeyUsage 成功使用后原子递增计数（Redis + DB 异步对账）。
func (s *APIKeyService) IncrKeyUsage(ctx context.Context, keyID uint64) {
	_ = s.redis.Incr(ctx, "apikey:used:"+fmt.Sprint(keyID)).Err()
	s.db.WithContext(ctx).Model(&model.ApiKey{}).Where("id = ?", keyID).UpdateColumns(map[string]any{
		"used":         gorm.Expr("used + 1"),
		"last_used_at": time.Now(),
	})
}

// ---- 工具函数 ----

func SHA256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func HMACSHA256Hex(secret, data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func ipInWhitelist(ip string, whitelist string) bool {
	parsedIP := net.ParseIP(ip)
	for _, entry := range strings.Split(whitelist, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err == nil && cidr.Contains(parsedIP) {
				return true
			}
		} else if entry == ip {
			return true
		}
	}
	return false
}

const tokenCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// generateToken 生成 prefix + n 位密码学安全 Base62 随机字符串。
func generateToken(prefix string, n int) (string, error) {
	b := make([]byte, n)
	max := big.NewInt(int64(len(tokenCharset)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = tokenCharset[idx.Int64()]
	}
	return prefix + string(b), nil
}
