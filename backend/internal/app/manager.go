package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/database"
	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/router"
	"gravitylink/backend/internal/service"
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
	Database      databaseInput `json:"database"`
	Redis         redisInput    `json:"redis"`
	Auth          authInput     `json:"auth"`
	PublicBaseURL string        `json:"public_base_url"`
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
	Mode         string   `json:"mode"`
	Issuer       string   `json:"issuer"`
	AppID        string   `json:"app_id"`
	Audience     string   `json:"audience"`
	JWKSURL      string   `json:"jwks_url"`
	Scopes       string   `json:"scopes"`
	AdminBaseURL string   `json:"admin_base_url"`
	AllowedRoles []string `json:"allowed_roles"`
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	Password     string   `json:"password"`
}

type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

func NewManager(cfg config.Config, logger *slog.Logger) *Manager {
	manager := &Manager{cfg: cfg, logger: logger, reason: "configuration is incomplete"}
	manager.handler.Store(handlerHolder{handler: http.HandlerFunc(publicSetupRequired)})
	return manager
}

func (m *Manager) Start() error {
	return m.activate(m.cfg, false)
}

// RecoverStartup retries an installed runtime without persisting configuration.
// It exits permanently on success, reset, or shutdown.
func (m *Manager) RecoverStartup(ctx context.Context) {
	retryStartup(ctx, 5*time.Second, func() bool {
		m.mu.RLock()
		cfg, initialized := m.cfg, m.initialized
		m.mu.RUnlock()
		if initialized || !cfg.InstallationComplete || cfg.ResetPending {
			return true
		}
		return m.activate(cfg, false) == nil
	})
}

func retryStartup(ctx context.Context, interval time.Duration, attempt func() bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ctx.Err() != nil || attempt() {
				return
			}
		}
	}
}

func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.handler.Load().(handlerHolder).handler.ServeHTTP(w, r)
}

func (m *Manager) SetupHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/setup/status", m.setupStatus)
	mux.HandleFunc("POST /api/setup/auth/logto/check", m.checkLogto)
	mux.HandleFunc("POST /api/setup/auth/logto/claim", m.claimLogto)
	mux.HandleFunc("POST /api/setup/auth/local", m.configureLocalOwner)
	mux.HandleFunc("POST /api/setup/database/test", m.testDatabase)
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

func (m *Manager) Reset(ctx context.Context, userID uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.initialized || m.db == nil {
		return fmt.Errorf("system is not initialized")
	}
	if err := config.MarkReset(m.cfg.ConfigFile); err != nil {
		return fmt.Errorf("write reset marker: %w", err)
	}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		detail, _ := json.Marshal(map[string]any{"preserved_business_data": true})
		if err := tx.Create(&model.AuditLog{
			UserID: &userID, Action: "system.reset", Detail: detail,
		}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.AuthSession{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.SystemConfig{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&model.User{}).
			Updates(map[string]any{"role": model.UserRoleUser, "status": model.StatusDisabled}).Error; err != nil {
			return err
		}
		return tx.Model(&model.InstallationState{}).Where("id = ?", 1).Updates(map[string]any{
			"installed": false, "owner_user_id": nil, "installed_at": nil,
		}).Error
	})
	if err != nil {
		_ = config.ClearResetMarker(m.cfg.ConfigFile)
		return err
	}

	if m.stopWorkers != nil {
		m.stopWorkers()
		m.stopWorkers = nil
	}
	if m.redis != nil {
		_ = m.redis.Close()
		m.redis = nil
	}
	if m.db != nil {
		if sqlDB, dbErr := m.db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		m.db = nil
	}
	if err := config.RemoveFile(m.cfg.ConfigFile); err != nil {
		return fmt.Errorf("remove runtime config: %w", err)
	}
	next := config.Load()
	next.ResetPending = true
	next.InstallationComplete = false
	next.BootstrapVerified = false
	m.cfg = next
	m.initialized = false
	m.reason = "系统配置已清除，请重新初始化"
	m.handler.Store(handlerHolder{handler: http.HandlerFunc(publicSetupRequired)})
	m.logger.Warn("system configuration reset", "user_id", userID)
	return nil
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
	if err := database.EnsureSchema(db); err != nil {
		m.reason = "MySQL schema initialization failed: " + err.Error()
		return fmt.Errorf("initialize schema: %w", err)
	}
	authService := service.NewAuthService(db)
	if persist {
		if _, err := authService.CreateBootstrapOwner(cfg); err != nil {
			m.reason = "Create super administrator failed: " + err.Error()
			return fmt.Errorf("create super administrator: %w", err)
		}
		cfg.InstallationComplete = true
		cfg.BootstrapVerified = false
		cfg.BootstrapSubject = ""
		cfg.BootstrapUsername = ""
		cfg.BootstrapEmail = ""
		cfg.BootstrapPassword = ""
	} else {
		installed, err := authService.Installed()
		if err != nil || !installed {
			m.reason = "Administrator initialization is incomplete"
			return fmt.Errorf("administrator initialization is incomplete")
		}
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

	workerCtx, stopWorkers := context.WithCancel(context.Background())
	notifier := service.NewNotifier(db, redisClient)
	go worker.NewAccessLogConsumer(db, redisClient, m.logger).Start(workerCtx)
	go worker.NewLogArchiver(db, m.logger).Start(workerCtx)
	go worker.NewStatFlusher(db, redisClient, m.logger).Start(workerCtx)
	go worker.NewDomainCheckWorker(db, redisClient, notifier, m.logger).Start(workerCtx)

	engine := router.New(router.Dependencies{
		Config: cfg, DB: db, Redis: redisClient, Logger: m.logger, ResetSystem: m.Reset, Notifier: notifier,
	})
	if persist {
		if err := config.SaveFile(cfg.ConfigFile, cfg); err != nil {
			m.reason = "Failed to save configuration: " + err.Error()
			return fmt.Errorf("save config: %w", err)
		}
	}

	m.cfg = cfg
	m.db = db
	m.redis = redisClient
	m.stopWorkers = stopWorkers
	m.initialized = true
	m.reason = ""
	m.handler.Store(handlerHolder{handler: engine})
	closeDB = false
	closeRedis = false
	m.logger.Info("gravitylink runtime initialized", "auth_mode", cfg.AuthMode)
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
			"mode": m.cfg.AuthMode, "issuer": m.cfg.LogtoIssuer, "app_id": m.cfg.LogtoAppID,
			"audience": m.cfg.LogtoAudience, "jwks_url": m.cfg.LogtoJWKSURL, "scopes": m.cfg.LogtoScopes,
			"admin_base_url": m.cfg.AdminBaseURL, "allowed_roles": m.cfg.AdminAllowedRoles,
			"verified": m.cfg.BootstrapVerified, "owner_username": m.cfg.BootstrapUsername,
			"owner_email": m.cfg.BootstrapEmail,
		},
	})
}

func (m *Manager) checkLogto(w http.ResponseWriter, r *http.Request) {
	if m.isInitialized() {
		writeError(w, http.StatusConflict, 4409, "setup_locked")
		return
	}
	var input setupRequest
	if err := decodeRequest(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, 4000, err.Error())
		return
	}
	cfg := m.currentConfig()
	applyAuthInput(&cfg, input.Auth)
	cfg.AuthMode = model.AuthSourceLogto
	if cfg.LogtoIssuer == "" || cfg.LogtoAppID == "" || cfg.LogtoAudience == "" || cfg.AdminBaseURL == "" {
		writeError(w, http.StatusBadRequest, 4004, "Logto Issuer、App ID、Audience 和管理端 URL 均为必填项")
		return
	}

	discoveryURL := strings.TrimRight(cfg.LogtoIssuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, discoveryURL, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, 4004, err.Error())
		return
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Warn("logto discovery request failed", "url", discoveryURL, "error", err)
		writeError(w, http.StatusBadRequest, 4004, "无法连接 Logto："+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadRequest, 4004, fmt.Sprintf("Logto Discovery 返回 HTTP %d", resp.StatusCode))
		return
	}
	var discovery oidcDiscovery
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&discovery); err != nil {
		writeError(w, http.StatusBadRequest, 4004, "Logto Discovery 响应无效")
		return
	}
	if discovery.AuthorizationEndpoint == "" || discovery.TokenEndpoint == "" || discovery.JWKSURI == "" {
		writeError(w, http.StatusBadRequest, 4004, "Logto Discovery 缺少必要端点")
		return
	}
	if cfg.LogtoJWKSURL == "" {
		cfg.LogtoJWKSURL = discovery.JWKSURI
	}
	cfg.BootstrapVerified = false
	if err := m.saveDraft(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, 5000, "保存 Logto 配置失败："+err.Error())
		return
	}
	writeOK(w, map[string]any{
		"issuer": discovery.Issuer, "authorization_endpoint": discovery.AuthorizationEndpoint,
		"token_endpoint": discovery.TokenEndpoint, "jwks_uri": discovery.JWKSURI,
		"redirect_uri": strings.TrimRight(cfg.AdminBaseURL, "/") + "/setup/auth/callback",
	})
}

func (m *Manager) claimLogto(w http.ResponseWriter, r *http.Request) {
	if m.isInitialized() {
		writeError(w, http.StatusConflict, 4409, "setup_locked")
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		writeError(w, http.StatusUnauthorized, 4401, "missing Logto access token")
		return
	}
	cfg := m.currentConfig()
	if cfg.AuthMode != model.AuthSourceLogto {
		writeError(w, http.StatusBadRequest, 4004, "Logto 尚未配置")
		return
	}
	identity, err := middleware.VerifyToken(r.Context(), cfg, token)
	if err != nil || identity.Subject == "" {
		m.logger.Warn("logto claim failed", "error", err, "subject", identity.Subject, "issuer", cfg.LogtoIssuer, "audience", cfg.LogtoAudience, "jwks_url", cfg.LogtoJWKSURL)
		writeError(w, http.StatusUnauthorized, 4401, "Logto 身份验证失败")
		return
	}
	cfg.BootstrapVerified = true
	cfg.BootstrapSubject = identity.Subject
	cfg.BootstrapUsername = firstNonEmpty(identity.Username, identity.Email, "owner")
	cfg.BootstrapEmail = identity.Email
	cfg.BootstrapPassword = ""
	if err := m.saveDraft(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, 5000, "保存管理员身份失败："+err.Error())
		return
	}
	writeOK(w, map[string]any{"verified": true, "username": cfg.BootstrapUsername, "email": cfg.BootstrapEmail})
}

func (m *Manager) configureLocalOwner(w http.ResponseWriter, r *http.Request) {
	if m.isInitialized() {
		writeError(w, http.StatusConflict, 4409, "setup_locked")
		return
	}
	var input setupRequest
	if err := decodeRequest(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, 4000, err.Error())
		return
	}
	username := strings.TrimSpace(input.Auth.Username)
	if len(username) < 3 {
		writeError(w, http.StatusBadRequest, 4005, "用户名至少需要 3 个字符")
		return
	}
	passwordHash, err := service.HashPassword(input.Auth.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, 4005, "密码至少需要 10 个字符")
		return
	}
	cfg := m.currentConfig()
	cfg.AuthMode = model.AuthSourceLocal
	cfg.BootstrapVerified = true
	cfg.BootstrapSubject = ""
	cfg.BootstrapUsername = username
	cfg.BootstrapEmail = strings.TrimSpace(input.Auth.Email)
	cfg.BootstrapPassword = passwordHash
	if input.Auth.AdminBaseURL != "" {
		cfg.AdminBaseURL = strings.TrimRight(strings.TrimSpace(input.Auth.AdminBaseURL), "/")
	}
	if err := m.saveDraft(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, 5000, "保存本地管理员配置失败："+err.Error())
		return
	}
	writeOK(w, map[string]any{"verified": true, "username": username, "email": cfg.BootstrapEmail})
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

	// 测试通过后保存 draft，使数据库密码在 Logto OAuth 页面跳转后不丢失
	if err := m.saveDraft(cfg); err != nil {
		m.logger.Warn("save database draft failed", "error", err)
	}
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
		m.logger.Error("setup activation failed", "error", err)
		writeError(w, http.StatusBadRequest, 4003, friendlySetupError(err))
		return
	}
	// 初始化完成后写入公开访问地址等系统配置
	if pubURL := strings.TrimRight(strings.TrimSpace(input.PublicBaseURL), "/"); pubURL != "" && m.db != nil {
		configSvc := service.NewSystemConfigService(m.db)
		_, _ = configSvc.Update(r.Context(), service.SystemConfigInput{
			Configs: map[string]string{"public.base_url": pubURL},
		})
	}
	writeOK(w, map[string]any{"initialized": true})
}

// friendlySetupError 将 activate 阶段的底层错误映射为用户可理解的中文提示，
// 原始错误已记录到服务端日志，不再直接透出 SQL 细节。
func friendlySetupError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "connect mysql"):
		return "无法连接 MySQL，请返回数据服务步骤检查连接配置"
	case strings.Contains(msg, "initialize schema"), strings.Contains(msg, "数据库表结构"):
		return "数据库结构初始化失败：" + msg
	case strings.Contains(msg, "create super administrator"):
		return "创建超级管理员失败，请返回管理员身份步骤重新填写后重试"
	case strings.Contains(msg, "administrator initialization"):
		return "管理员身份未完成验证，请返回管理员身份步骤"
	case strings.Contains(msg, "connect redis"):
		return "无法连接 Redis，请返回数据服务步骤检查连接配置"
	case strings.Contains(msg, "save config"):
		return "保存配置文件失败，请检查服务端 /data 目录权限后重试"
	case strings.Contains(msg, "生产环境"), strings.Contains(msg, "请先完成"), strings.Contains(msg, "配置不完整"), strings.Contains(msg, "重新初始化"):
		return msg
	default:
		return "初始化失败，请检查各项配置后重试（详细信息请查看服务端日志）"
	}
}

func (m *Manager) configFromInput(input setupRequest) config.Config {
	m.mu.RLock()
	cfg := m.cfg
	m.mu.RUnlock()

	cfg.MySQLHost = strings.TrimSpace(input.Database.Host)
	cfg.MySQLPort = strings.TrimSpace(input.Database.Port)
	cfg.MySQLDatabase = strings.TrimSpace(input.Database.Database)
	cfg.MySQLUser = strings.TrimSpace(input.Database.User)
	// 空密码时保留 draft 中已保存的值（Logto OAuth 页面跳转会丢失前端表单状态）
	if input.Database.Password != "" {
		cfg.MySQLPassword = input.Database.Password
	}
	// params 为空时保留默认值（charset/parseTime 等），避免拼出无参数的 DSN
	if params := strings.TrimSpace(input.Database.Params); params != "" {
		cfg.MySQLParams = params
	}
	cfg.MySQLDSN = strings.TrimSpace(input.Database.DSN)
	cfg.RedisHost = strings.TrimSpace(input.Redis.Host)
	cfg.RedisPort = strings.TrimSpace(input.Redis.Port)
	cfg.RedisAddr = strings.TrimSpace(input.Redis.Addr)
	if input.Redis.Password != "" {
		cfg.RedisPassword = input.Redis.Password
	}
	cfg.RedisDB = input.Redis.DB
	if input.Auth.Mode != "" || input.Auth.Issuer != "" {
		applyAuthInput(&cfg, input.Auth)
	}
	cfg.RebuildConnections()
	return cfg
}

func (m *Manager) isInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

func (m *Manager) currentConfig() config.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) saveDraft(cfg config.Config) error {
	cfg.ResetPending = false
	if err := config.SaveFile(cfg.ConfigFile, cfg); err != nil {
		return err
	}
	if err := config.ClearResetMarker(cfg.ConfigFile); err != nil {
		return err
	}
	m.mu.Lock()
	m.cfg = cfg
	m.reason = ""
	m.mu.Unlock()
	return nil
}

func validateConfig(cfg config.Config) error {
	if cfg.ResetPending {
		return fmt.Errorf("系统配置已清除，请重新完成初始化")
	}
	if !cfg.InstallationComplete && !cfg.BootstrapVerified {
		return fmt.Errorf("请先完成管理员身份验证")
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

func applyAuthInput(cfg *config.Config, input authInput) {
	if input.Mode != "" {
		cfg.AuthMode = strings.TrimSpace(input.Mode)
	}
	cfg.LogtoIssuer = strings.TrimRight(strings.TrimSpace(input.Issuer), "/")
	cfg.LogtoAppID = strings.TrimSpace(input.AppID)
	cfg.LogtoAudience = strings.TrimSpace(input.Audience)
	cfg.LogtoJWKSURL = strings.TrimSpace(input.JWKSURL)
	cfg.LogtoScopes = strings.TrimSpace(input.Scopes)
	if cfg.LogtoScopes == "" {
		cfg.LogtoScopes = "openid profile email"
	}
	cfg.AdminBaseURL = strings.TrimRight(strings.TrimSpace(input.AdminBaseURL), "/")
	cfg.AdminAllowedRoles = cleanList(input.AllowedRoles)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func decodeRequest(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
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
	_, _ = w.Write([]byte(`<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>GravityLink 尚未初始化</title><style>body{margin:0;min-height:100vh;display:grid;place-items:center;font-family:system-ui,sans-serif;background:#f4f7f8;color:#172026}.box{max-width:560px;padding:32px}.mark{width:44px;height:44px;display:grid;place-items:center;border-radius:8px;background:#188d7c;color:white;font-weight:800}h1{font-size:26px}p{color:#60727a;line-height:1.7}</style><main class="box"><div class="mark">G</div><h1>GravityLink 尚未初始化</h1><p>请由管理员访问管理端口，完成管理员身份、数据库与 Redis 配置。</p></main></html>`))
}
