package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gravitylink/backend/internal/model"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrWechat = errors.New("微信分享暂不可用，请检查公众号配置、IP 白名单和安全域名")

type WechatService struct {
	db      *gorm.DB
	keyFile string
	mu      sync.Mutex
	ticket  string
	expires time.Time
	client  *http.Client
}

func NewWechatService(db *gorm.DB, dir string) *WechatService {
	return &WechatService{db: db, keyFile: filepath.Join(dir, "wechat.key"), client: &http.Client{Timeout: 8 * time.Second}}
}
func (s *WechatService) key() ([]byte, error) {
	b, e := os.ReadFile(s.keyFile)
	if e == nil {
		if len(b) != 32 {
			return nil, ErrWechat
		}
		return b, nil
	}
	if !os.IsNotExist(e) {
		return nil, e
	}
	if e = os.MkdirAll(filepath.Dir(s.keyFile), 0700); e != nil {
		return nil, e
	}
	b = make([]byte, 32)
	if _, e = rand.Read(b); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(s.keyFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	_, e = f.Write(b)
	return b, e
}
func sealSecret(key []byte, value string) (string, error) {
	block, e := aes.NewCipher(key)
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}
	nonce := make([]byte, g.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	return base64.StdEncoding.EncodeToString(g.Seal(nonce, nonce, []byte(value), nil)), nil
}
func openSecret(key []byte, value string) (string, error) {
	b, e := base64.StdEncoding.DecodeString(value)
	if e != nil {
		return "", ErrWechat
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(block)
	if e != nil || len(b) < g.NonceSize() {
		return "", ErrWechat
	}
	plain, e := g.Open(nil, b[:g.NonceSize()], b[g.NonceSize():], nil)
	return string(plain), e
}
func (s *WechatService) Status(ctx context.Context) (string, bool, error) {
	var a model.WechatAccount
	e := s.db.WithContext(ctx).First(&a, 1).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	return a.AppID, a.Secret != "", e
}
func (s *WechatService) Save(ctx context.Context, appID, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !strings.HasPrefix(appID, "wx") || len(appID) > 128 {
		return ErrWechat
	}
	var a model.WechatAccount
	e := s.db.WithContext(ctx).First(&a, 1).Error
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return e
	}
	if secret == "" && (a.Secret == "" || a.AppID != appID) {
		return errors.New("首次配置或更换 AppID 时必须填写 AppSecret")
	}
	if secret != "" {
		key, e := s.key()
		if e != nil {
			return e
		}
		a.Secret, e = sealSecret(key, secret)
		if e != nil {
			return e
		}
	}
	a.ID = 1
	a.AppID = appID
	if e = s.db.WithContext(ctx).Save(&a).Error; e != nil {
		return e
	}
	s.ticket = ""
	return nil
}
func (s *WechatService) fetch(ctx context.Context, address string, out any) error {
	req, e := http.NewRequestWithContext(ctx, "GET", address, nil)
	if e != nil {
		return ErrWechat
	}
	r, e := s.client.Do(req)
	if e != nil {
		return ErrWechat
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return ErrWechat
	}
	if json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(out) != nil {
		return ErrWechat
	}
	return nil
}
func WechatSignature(ticket, nonce string, timestamp int64, pageURL string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", ticket, nonce, timestamp, strings.Split(pageURL, "#")[0])))
	return hex.EncodeToString(sum[:])
}
func (s *WechatService) Sign(ctx context.Context, pageURL string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var a model.WechatAccount
	if s.db.WithContext(ctx).First(&a, 1).Error != nil {
		return nil, ErrWechat
	}
	if s.ticket == "" || time.Now().After(s.expires) {
		key, e := s.key()
		if e != nil {
			return nil, ErrWechat
		}
		secret, e := openSecret(key, a.Secret)
		if e != nil {
			return nil, ErrWechat
		}
		var token struct {
			AccessToken string `json:"access_token"`
		}
		if s.fetch(ctx, "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid="+url.QueryEscape(a.AppID)+"&secret="+url.QueryEscape(secret), &token) != nil || token.AccessToken == "" {
			return nil, ErrWechat
		}
		var ticket struct {
			Ticket  string `json:"ticket"`
			Expires int    `json:"expires_in"`
		}
		if s.fetch(ctx, "https://api.weixin.qq.com/cgi-bin/ticket/getticket?type=jsapi&access_token="+url.QueryEscape(token.AccessToken), &ticket) != nil || ticket.Ticket == "" || ticket.Expires <= 120 {
			return nil, ErrWechat
		}
		s.ticket = ticket.Ticket
		s.expires = time.Now().Add(time.Duration(ticket.Expires-120) * time.Second)
	}
	nonce := make([]byte, 16)
	if _, e := rand.Read(nonce); e != nil {
		return nil, ErrWechat
	}
	n := hex.EncodeToString(nonce)
	ts := time.Now().Unix()
	return map[string]any{"appId": a.AppID, "timestamp": ts, "nonceStr": n, "signature": WechatSignature(s.ticket, n, ts, pageURL)}, nil
}
