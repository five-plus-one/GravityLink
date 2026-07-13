package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv        string
	HTTPAddr      string
	LogLevel      slog.Level
	MySQLDSN      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	AuthDisabled  bool
	LogtoIssuer   string
	LogtoAudience string
	LogtoJWKSURL  string
}

func Load() Config {
	return Config{
		AppEnv:        envString("APP_ENV", "development"),
		HTTPAddr:      envString("HTTP_ADDR", ":8080"),
		LogLevel:      envLogLevel("LOG_LEVEL", slog.LevelInfo),
		MySQLDSN:      envString("MYSQL_DSN", "gravitylink:gravitylink@tcp(127.0.0.1:3306)/gravitylink?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisAddr:     envString("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: envString("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),
		AuthDisabled:  envBool("AUTH_DISABLED", false),
		LogtoIssuer:   envString("LOGTO_ISSUER", ""),
		LogtoAudience: envString("LOGTO_AUDIENCE", ""),
		LogtoJWKSURL:  envString("LOGTO_JWKS_URL", ""),
	}
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
