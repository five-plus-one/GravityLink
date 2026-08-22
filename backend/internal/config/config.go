package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv               string
	HTTPAddr             string
	AdminHTTPAddr        string
	LogLevel             slog.Level
	ConfigFile           string
	MySQLHost            string
	MySQLPort            string
	MySQLDatabase        string
	MySQLUser            string
	MySQLPassword        string
	MySQLParams          string
	MySQLDSN             string
	RedisHost            string
	RedisPort            string
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	AuthMode             string
	LogtoIssuer          string
	LogtoAppID           string
	LogtoAudience        string
	LogtoJWKSURL         string
	LogtoScopes          string
	AdminBaseURL         string
	AdminAllowedRoles    []string
	BootstrapVerified    bool
	BootstrapSubject     string
	BootstrapUsername    string
	BootstrapEmail       string
	BootstrapPassword    string
	ResetPending         bool
	InstallationComplete bool
}

type FileConfig struct {
	AppEnv               string   `json:"app_env"`
	MySQLHost            string   `json:"mysql_host"`
	MySQLPort            string   `json:"mysql_port"`
	MySQLDatabase        string   `json:"mysql_database"`
	MySQLUser            string   `json:"mysql_user"`
	MySQLPassword        string   `json:"mysql_password"`
	MySQLParams          string   `json:"mysql_params"`
	MySQLDSN             string   `json:"mysql_dsn,omitempty"`
	RedisHost            string   `json:"redis_host"`
	RedisPort            string   `json:"redis_port"`
	RedisAddr            string   `json:"redis_addr,omitempty"`
	RedisPassword        string   `json:"redis_password,omitempty"`
	RedisDB              int      `json:"redis_db"`
	AuthMode             string   `json:"auth_mode"`
	LogtoIssuer          string   `json:"logto_issuer,omitempty"`
	LogtoAppID           string   `json:"logto_app_id,omitempty"`
	LogtoAudience        string   `json:"logto_audience,omitempty"`
	LogtoJWKSURL         string   `json:"logto_jwks_url,omitempty"`
	LogtoScopes          string   `json:"logto_scopes"`
	AdminBaseURL         string   `json:"admin_base_url,omitempty"`
	AdminAllowedRoles    []string `json:"admin_allowed_roles"`
	BootstrapVerified    bool     `json:"bootstrap_verified,omitempty"`
	BootstrapSubject     string   `json:"bootstrap_subject,omitempty"`
	BootstrapUsername    string   `json:"bootstrap_username,omitempty"`
	BootstrapEmail       string   `json:"bootstrap_email,omitempty"`
	BootstrapPassword    string   `json:"bootstrap_password_hash,omitempty"`
	ResetPending         bool     `json:"reset_pending,omitempty"`
	InstallationComplete bool     `json:"installation_complete,omitempty"`
}

func Load() Config {
	cfg := Config{
		AppEnv:            "production",
		HTTPAddr:          ":8080",
		AdminHTTPAddr:     ":8081",
		LogLevel:          slog.LevelInfo,
		ConfigFile:        envString("CONFIG_FILE", "data/gravitylink.json"),
		MySQLHost:         envString("SETUP_DEFAULT_MYSQL_HOST", "127.0.0.1"),
		MySQLPort:         "3306",
		MySQLDatabase:     "gravitylink",
		MySQLUser:         "gravitylink",
		MySQLPassword:     "",
		MySQLParams:       "charset=utf8mb4&parseTime=True&loc=Local",
		RedisHost:         envString("SETUP_DEFAULT_REDIS_HOST", "127.0.0.1"),
		RedisPort:         "6379",
		RedisDB:           0,
		AuthMode:          "logto",
		LogtoScopes:       "openid profile email",
		AdminAllowedRoles: []string{"admin"},
	}
	if persisted, err := loadFile(cfg.ConfigFile); err == nil {
		applyFile(&cfg, persisted)
	}
	applyEnvironment(&cfg)
	cfg.rebuildConnections()
	if _, err := os.Stat(resetMarkerPath(cfg.ConfigFile)); err == nil {
		cfg.ResetPending = true
		cfg.InstallationComplete = false
	}
	return cfg
}

func SaveFile(path string, cfg Config) error {
	if path == "" {
		return fmt.Errorf("config file path is empty")
	}
	payload, err := json.MarshalIndent(fileFromConfig(cfg), "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".gravitylink-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return os.Rename(tempPath, path)
	}
	backupPath := path + ".bak"
	_ = os.Remove(backupPath)
	if err := os.Rename(path, backupPath); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Rename(backupPath, path)
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}

func MarkReset(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(resetMarkerPath(path), []byte("reset_pending\n"), 0o600)
}

func ClearResetMarker(path string) error {
	err := os.Remove(resetMarkerPath(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func RemoveFile(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func resetMarkerPath(path string) string {
	return path + ".reset"
}

func (c Config) MissingRuntimeConfig() []string {
	var missing []string
	if c.MySQLDSN == "" {
		missing = append(missing, "mysql")
	}
	if c.RedisAddr == "" {
		missing = append(missing, "redis")
	}
	if c.AuthMode != "local" {
		if c.LogtoIssuer == "" {
			missing = append(missing, "logto_issuer")
		}
		if c.LogtoAppID == "" {
			missing = append(missing, "logto_app_id")
		}
		if c.LogtoAudience == "" {
			missing = append(missing, "logto_audience")
		}
		if c.AdminBaseURL == "" {
			missing = append(missing, "admin_base_url")
		}
	}
	return missing
}

func (c *Config) RebuildConnections() {
	c.rebuildConnections()
}

func (c *Config) rebuildConnections() {
	if c.MySQLDSN == "" && c.MySQLHost != "" && c.MySQLPort != "" && c.MySQLDatabase != "" && c.MySQLUser != "" {
		if c.MySQLParams == "" {
			c.MySQLParams = "charset=utf8mb4&parseTime=True&loc=Local"
		}
		c.MySQLDSN = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", c.MySQLUser, c.MySQLPassword, c.MySQLHost, c.MySQLPort, c.MySQLDatabase, c.MySQLParams)
	}
	if c.RedisAddr == "" && c.RedisHost != "" && c.RedisPort != "" {
		c.RedisAddr = fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
	}
}

func loadFile(path string) (FileConfig, error) {
	var cfg FileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func applyFile(cfg *Config, file FileConfig) {
	cfg.AppEnv = first(file.AppEnv, cfg.AppEnv)
	cfg.MySQLHost = first(file.MySQLHost, cfg.MySQLHost)
	cfg.MySQLPort = first(file.MySQLPort, cfg.MySQLPort)
	cfg.MySQLDatabase = first(file.MySQLDatabase, cfg.MySQLDatabase)
	cfg.MySQLUser = first(file.MySQLUser, cfg.MySQLUser)
	cfg.MySQLPassword = file.MySQLPassword
	cfg.MySQLParams = first(file.MySQLParams, cfg.MySQLParams)
	cfg.MySQLDSN = file.MySQLDSN
	cfg.RedisHost = first(file.RedisHost, cfg.RedisHost)
	cfg.RedisPort = first(file.RedisPort, cfg.RedisPort)
	cfg.RedisAddr = file.RedisAddr
	cfg.RedisPassword = file.RedisPassword
	cfg.RedisDB = file.RedisDB
	cfg.AuthMode = first(file.AuthMode, cfg.AuthMode)
	cfg.LogtoIssuer = file.LogtoIssuer
	cfg.LogtoAppID = file.LogtoAppID
	cfg.LogtoAudience = file.LogtoAudience
	cfg.LogtoJWKSURL = file.LogtoJWKSURL
	cfg.LogtoScopes = first(file.LogtoScopes, cfg.LogtoScopes)
	cfg.AdminBaseURL = file.AdminBaseURL
	if len(file.AdminAllowedRoles) > 0 {
		cfg.AdminAllowedRoles = file.AdminAllowedRoles
	}
	cfg.BootstrapVerified = file.BootstrapVerified
	cfg.BootstrapSubject = file.BootstrapSubject
	cfg.BootstrapUsername = file.BootstrapUsername
	cfg.BootstrapEmail = file.BootstrapEmail
	cfg.BootstrapPassword = file.BootstrapPassword
	cfg.ResetPending = file.ResetPending
	cfg.InstallationComplete = file.InstallationComplete
}

func applyEnvironment(cfg *Config) {
	setString := func(key string, target *string) {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			*target = value
		}
	}
	setString("APP_ENV", &cfg.AppEnv)
	setString("HTTP_ADDR", &cfg.HTTPAddr)
	setString("ADMIN_HTTP_ADDR", &cfg.AdminHTTPAddr)
	setString("MYSQL_HOST", &cfg.MySQLHost)
	setString("MYSQL_PORT", &cfg.MySQLPort)
	setString("MYSQL_DATABASE", &cfg.MySQLDatabase)
	setString("MYSQL_USER", &cfg.MySQLUser)
	setString("MYSQL_PASSWORD", &cfg.MySQLPassword)
	setString("MYSQL_PARAMS", &cfg.MySQLParams)
	setString("MYSQL_DSN", &cfg.MySQLDSN)
	setString("REDIS_HOST", &cfg.RedisHost)
	setString("REDIS_PORT", &cfg.RedisPort)
	setString("REDIS_ADDR", &cfg.RedisAddr)
	setString("REDIS_PASSWORD", &cfg.RedisPassword)
	setString("LOGTO_ISSUER", &cfg.LogtoIssuer)
	setString("AUTH_MODE", &cfg.AuthMode)
	setString("LOGTO_APP_ID", &cfg.LogtoAppID)
	setString("LOGTO_AUDIENCE", &cfg.LogtoAudience)
	setString("LOGTO_JWKS_URL", &cfg.LogtoJWKSURL)
	setString("LOGTO_SCOPES", &cfg.LogtoScopes)
	setString("ADMIN_BASE_URL", &cfg.AdminBaseURL)
	if _, dsnSet := os.LookupEnv("MYSQL_DSN"); !dsnSet && anyEnvironmentSet(
		"MYSQL_HOST", "MYSQL_PORT", "MYSQL_DATABASE", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_PARAMS",
	) {
		cfg.MySQLDSN = ""
	}
	if _, addrSet := os.LookupEnv("REDIS_ADDR"); !addrSet && anyEnvironmentSet("REDIS_HOST", "REDIS_PORT") {
		cfg.RedisAddr = ""
	}
	cfg.LogLevel = envLogLevel("LOG_LEVEL", cfg.LogLevel)
	cfg.RedisDB = envInt("REDIS_DB", cfg.RedisDB)
	cfg.AdminAllowedRoles = envList("ADMIN_ALLOWED_ROLES", cfg.AdminAllowedRoles)
}

func anyEnvironmentSet(keys ...string) bool {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			return true
		}
	}
	return false
}

func fileFromConfig(cfg Config) FileConfig {
	return FileConfig{
		AppEnv: cfg.AppEnv, MySQLHost: cfg.MySQLHost, MySQLPort: cfg.MySQLPort,
		MySQLDatabase: cfg.MySQLDatabase, MySQLUser: cfg.MySQLUser, MySQLPassword: cfg.MySQLPassword,
		MySQLParams: cfg.MySQLParams, MySQLDSN: cfg.MySQLDSN, RedisHost: cfg.RedisHost,
		RedisPort: cfg.RedisPort, RedisAddr: cfg.RedisAddr, RedisPassword: cfg.RedisPassword,
		RedisDB: cfg.RedisDB, LogtoIssuer: cfg.LogtoIssuer,
		LogtoAppID: cfg.LogtoAppID, LogtoAudience: cfg.LogtoAudience, LogtoJWKSURL: cfg.LogtoJWKSURL,
		LogtoScopes: cfg.LogtoScopes, AdminBaseURL: cfg.AdminBaseURL, AdminAllowedRoles: cfg.AdminAllowedRoles,
		AuthMode: cfg.AuthMode, BootstrapVerified: cfg.BootstrapVerified, BootstrapSubject: cfg.BootstrapSubject,
		BootstrapUsername: cfg.BootstrapUsername, BootstrapEmail: cfg.BootstrapEmail,
		BootstrapPassword: cfg.BootstrapPassword, ResetPending: cfg.ResetPending,
		InstallationComplete: cfg.InstallationComplete,
	}
}

func first(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func envString(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	case "0", "false", "FALSE", "no", "NO":
		return false
	default:
		return fallback
	}
}

func envList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func envLogLevel(key string, fallback slog.Level) slog.Level {
	switch os.Getenv(key) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info":
		return slog.LevelInfo
	default:
		return fallback
	}
}

func (c Config) Now() time.Time {
	return time.Now().UTC()
}
