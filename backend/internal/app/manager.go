package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/database"
	"gravitylink/backend/internal/router"
	"gravitylink/backend/internal/worker"
)

type handlerHolder struct {
	handler http.Handler
}

type Manager struct {
	logger *slog.Logger

	mu          sync.RWMutex
	cfg         config.Config
	initialized bool
	reason      string
	handler     atomic.Value
	db          *gorm.DB
	redis       *redis.Client
	stopWorkers context.CancelFunc
}

type setupRequest struct {
	Database databaseInput `json:"database"`
	Redis    redisInput    `json:"redis"`
	Auth     authInput     `json:"auth"`
}

type databaseInput struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
	Params   string `json:"params"`
	DSN      string `json:"dsn"`
}

type redisInput struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type authInput struct {
	Disabled     bool     `json:"disabled"`
	Issuer       string   `json:"issuer"`
	AppID        string   `json:"app_id"`
	Audience     string   `json:"audience"`
	JWKSURL      string   `json:"jwks_url"`
	Scopes       string   `json:"scopes"`
	AdminBaseURL string   `json:"admin_base_url"`
	AllowedRoles []string `json:"allowed_roles"`
}

func NewManager(cfg config.Config, logger *slog.Logger) *Manager {
	manager := &Manager{cfg: cfg, logger: logger, reason: "configuration is incomplete"}
	manager.handler.Store(handlerHolder{handler: http.HandlerFunc(publicSetupRequired)})
	return manager
}

func (m *Manager) Start() error {
	return m.activate(m.cfg, false)
}

func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.handler.Load().(handlerHolder).handler.ServeHTTP(w, r)
}

func (m *Manager) SetupHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/setup/status", m.setupStatus)
	mux.HandleFunc("POST /api/setup/test-database", m.testDatabase)
	mux.HandleFunc("POST /api/setup/complete", m.completeSetup)
	return mux
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopWorkers != nil {
		m.stopWorkers()
	}
	if m.redis != nil {
		if err := m.redis.Close(); err != nil {
			m.logger.Warn("close redis failed", "error", err)
		}
	}
	if m.db != nil {
		if sqlDB, err := m.db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				m.logger.Warn("close mysql failed", "error", err)
			}
		}
	}
}

func (m *Manager) activate(cfg config.Config, persist bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.initialized {
		return fmt.Errorf("setup is already complete")
	}
	if err := validateConfig(cfg); err != nil {
		m.reason = err.Error()
		return err
	}

	db, err := database.ConnectMySQL(cfg.MySQLDSN)
	if err != nil {
		m.reason = "MySQL connection failed: " + err.Error()
		return fmt.Errorf("connect mysql: %w", err)
	}
	closeDB := true
	defer func() {
		if closeDB {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
		}
	}()
	if !db.Migrator().HasTable("system_configs") || !db.Migrator().HasTable("links") {
		m.reason = "MySQL is reachable, but the GravityLink schema is missing"
		return fmt.Errorf("gravitylink database schema is not initialized")
	}

	redisClient, err := database.ConnectRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		m.reason = "Redis connection failed: " + err.Error()
		return fmt.Errorf("connect redis: %w", err)
	}
	closeRedis := true
	defer func() {
		if closeRedis {
			_ = redisClient.Close()
		}
	}()

	engine := router.New(router.Dependencies{Config: cfg, DB: db, Redis: redisClient, Logger: m.logger})
	if persist {
		if err := config.SaveFile(cfg.ConfigFile, cfg); err != nil {
			m.reason = "Failed to save configuration: " + err.Error()
			return fmt.Errorf("save config: %w", err)
		}
	}

	workerCtx, stopWorkers := context.WithCancel(context.Background())
	go worker.NewAccessLogConsumer(db, redisClient, m.logger).Start(workerCtx)
	go worker.NewLogArchiver(db, m.logger).Start(workerCtx)

	m.cfg = cfg
	m.db = db
	m.redis = redisClient
	m.stopWorkers = stopWorkers
	m.initialized = true
	m.reason = ""
	m.handler.Store(handlerHolder{handler: engine})
	closeDB = false
	closeRedis = false
	m.logger.Info("gravitylink runtime initialized", "auth_disabled", cfg.AuthDisabled)
	return nil
}

func (m *Manager) setupStatus(w http.ResponseWriter, _ *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	writeOK(w, map[string]any{
		"setup_required": !m.initialized,
		"reason":         m.reason,
		"environment":    m.cfg.AppEnv,
		"database": map[string]any{
			"host": m.cfg.MySQLHost, "port": m.cfg.MySQLPort, "database": m.cfg.MySQLDatabase,
			"user": m.cfg.MySQLUser, "params": m.cfg.MySQLParams, "dsn_configured": m.cfg.MySQLDSN != "",
		},
		"redis": map[string]any{
			"host": m.cfg.RedisHost, "port": m.cfg.RedisPort, "db": m.cfg.RedisDB,
			"addr_configured": m.cfg.RedisAddr != "",
		},
		"auth": map[string]any{
			"disabled": m.cfg.AuthDisabled, "issuer": m.cfg.LogtoIssuer, "app_id": m.cfg.LogtoAppID,
			"audience": m.cfg.LogtoAudience, "jwks_url": m.cfg.LogtoJWKSURL, "scopes": m.cfg.LogtoScopes,
			"admin_base_url": m.cfg.AdminBaseURL, "allowed_roles": m.cfg.AdminAllowedRoles,
		},
	})
}

func (m *Manager) testDatabase(w http.ResponseWriter, r *http.Request) {
	if m.isInitialized() {
		writeError(w, http.StatusConflict, 4409, "setup_locked")
		return
	}
	var input setupRequest
	if err := decodeRequest(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, 4000, err.Error())
		return
	}
	cfg := m.configFromInput(input)
	db, err := database.ConnectMySQL(cfg.MySQLDSN)
	if err != nil {
		writeError(w, http.StatusBadRequest, 4001, "MySQL 连接失败："+err.Error())
		return
	}
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		defer sqlDB.Close()
	}
	schemaReady := db.Migrator().HasTable("system_configs") && db.Migrator().HasTable("links")

	redisClient, err := database.ConnectRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		writeError(w, http.StatusBadRequest, 4002, "Redis 连接失败："+err.Error())
		return
	}
	defer redisClient.Close()
	writeOK(w, map[string]any{"mysql": true, "redis": true, "schema_ready": schemaReady})
}

func (m *Manager) completeSetup(w http.ResponseWriter, r *http.Request) {
	if m.isInitialized() {
		writeError(w, http.StatusConflict, 4409, "setup_locked")
		return
	}
	var input setupRequest
	if err := decodeRequest(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, 4000, err.Error())
		return
	}
	cfg := m.configFromInput(input)
	if err := m.activate(cfg, true); err != nil {
		writeError(w, http.StatusBadRequest, 4003, err.Error())
		return
	}
	writeOK(w, map[string]any{"initialized": true, "auth_disabled": cfg.AuthDisabled})
}

func (m *Manager) configFromInput(input setupRequest) config.Config {
	m.mu.RLock()
	cfg := m.cfg
	m.mu.RUnlock()

	cfg.MySQLHost = strings.TrimSpace(input.Database.Host)
	cfg.MySQLPort = strings.TrimSpace(input.Database.Port)
	cfg.MySQLDatabase = strings.TrimSpace(input.Database.Database)
	cfg.MySQLUser = strings.TrimSpace(input.Database.User)
	cfg.MySQLPassword = input.Database.Password
	cfg.MySQLParams = strings.TrimSpace(input.Database.Params)
	cfg.MySQLDSN = strings.TrimSpace(input.Database.DSN)
	cfg.RedisHost = strings.TrimSpace(input.Redis.Host)
	cfg.RedisPort = strings.TrimSpace(input.Redis.Port)
	cfg.RedisAddr = strings.TrimSpace(input.Redis.Addr)
	cfg.RedisPassword = input.Redis.Password
	cfg.RedisDB = input.Redis.DB
	cfg.AuthDisabled = input.Auth.Disabled
	cfg.LogtoIssuer = strings.TrimRight(strings.TrimSpace(input.Auth.Issuer), "/")
	cfg.LogtoAppID = strings.TrimSpace(input.Auth.AppID)
	cfg.LogtoAudience = strings.TrimSpace(input.Auth.Audience)
	cfg.LogtoJWKSURL = strings.TrimSpace(input.Auth.JWKSURL)
	cfg.LogtoScopes = strings.TrimSpace(input.Auth.Scopes)
	cfg.AdminBaseURL = strings.TrimRight(strings.TrimSpace(input.Auth.AdminBaseURL), "/")
	cfg.AdminAllowedRoles = cleanList(input.Auth.AllowedRoles)
	cfg.RebuildConnections()
	return cfg
}

func (m *Manager) isInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

func validateConfig(cfg config.Config) error {
	if cfg.AppEnv == "production" && cfg.AuthDisabled {
		return fmt.Errorf("生产环境不能关闭身份认证")
	}
	if missing := cfg.MissingRuntimeConfig(); len(missing) > 0 {
		return fmt.Errorf("配置不完整：%s", strings.Join(missing, ", "))
	}
	return nil
}

func cleanList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if item := strings.TrimSpace(part); item != "" {
				result = append(result, item)
			}
		}
	}
	if len(result) == 0 {
		return []string{"admin"}
	}
	return result
}

func decodeRequest(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("请求格式错误：%w", err)
	}
	return nil
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "message": "ok", "data": data})
}

func writeError(w http.ResponseWriter, status int, code int, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message, "data": nil})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func publicSetupRequired(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusServiceUnavailable, 4503, "setup_required")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>GravityLink 尚未初始化</title><style>body{margin:0;min-height:100vh;display:grid;place-items:center;font-family:system-ui,sans-serif;background:#f4f7f8;color:#172026}.box{max-width:560px;padding:32px}.mark{width:44px;height:44px;display:grid;place-items:center;border-radius:8px;background:#188d7c;color:white;font-weight:800}h1{font-size:26px}p{color:#60727a;line-height:1.7}</style><main class="box"><div class="mark">G</div><h1>GravityLink 尚未初始化</h1><p>请由管理员访问管理端口完成数据库、Redis 与 Logto 配置。</p></main></html>`))
}
