package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKeepsPersistedValuesWhenComposePassesEmptyEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gravitylink.json")
	persisted := Config{
		AppEnv: "production", ConfigFile: path,
		MySQLHost: "db.internal", MySQLPort: "3307", MySQLDatabase: "gravitylink",
		MySQLUser: "app", MySQLPassword: "secret", MySQLParams: "parseTime=True",
		RedisHost: "cache.internal", RedisPort: "6380", RedisPassword: "redis-secret", RedisDB: 2,
		LogtoIssuer: "https://login.example.com/oidc", LogtoAppID: "spa",
		LogtoAudience: "https://api.example.com", LogtoScopes: "openid profile",
		AdminBaseURL: "https://admin.example.com", AdminAllowedRoles: []string{"admin"},
	}
	persisted.RebuildConnections()
	if err := SaveFile(path, persisted); err != nil {
		t.Fatalf("save config: %v", err)
	}

	t.Setenv("CONFIG_FILE", path)
	for _, key := range []string{
		"MYSQL_HOST", "MYSQL_PORT", "MYSQL_DATABASE", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_PARAMS", "MYSQL_DSN",
		"REDIS_HOST", "REDIS_PORT", "REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"AUTH_DISABLED", "LOGTO_ISSUER", "LOGTO_APP_ID", "LOGTO_AUDIENCE", "LOGTO_SCOPES", "ADMIN_BASE_URL", "ADMIN_ALLOWED_ROLES",
	} {
		t.Setenv(key, "")
	}

	loaded := Load()
	if loaded.MySQLHost != persisted.MySQLHost || loaded.MySQLPassword != persisted.MySQLPassword {
		t.Fatalf("persisted mysql config was overwritten: %#v", loaded)
	}
	if loaded.RedisAddr != "cache.internal:6380" || loaded.RedisDB != 2 {
		t.Fatalf("persisted redis config was overwritten: %#v", loaded)
	}
	if loaded.LogtoIssuer != persisted.LogtoIssuer || loaded.LogtoAppID != persisted.LogtoAppID {
		t.Fatalf("persisted Logto config was overwritten: %#v", loaded)
	}
}

func TestLoadNonEmptyEnvironmentOverridesPersistedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gravitylink.json")
	persisted := Config{
		AppEnv: "production", ConfigFile: path,
		MySQLHost: "db.internal", MySQLPort: "3306", MySQLDatabase: "gravitylink",
		MySQLUser: "app", MySQLPassword: "secret", MySQLParams: "parseTime=True",
		RedisHost: "cache.internal", RedisPort: "6379",
		AuthDisabled: true, LogtoScopes: "openid", AdminAllowedRoles: []string{"admin"},
	}
	persisted.RebuildConnections()
	if err := SaveFile(path, persisted); err != nil {
		t.Fatalf("save config: %v", err)
	}

	t.Setenv("CONFIG_FILE", path)
	t.Setenv("MYSQL_HOST", "db.from-env")
	t.Setenv("MYSQL_PASSWORD", "env-secret")
	t.Setenv("REDIS_HOST", "redis.from-env")
	t.Setenv("AUTH_DISABLED", "false")
	t.Setenv("LOGTO_ISSUER", "https://env.example.com/oidc")

	loaded := Load()
	if loaded.MySQLHost != "db.from-env" || loaded.MySQLPassword != "env-secret" {
		t.Fatalf("environment did not override mysql config: %#v", loaded)
	}
	if loaded.RedisHost != "redis.from-env" {
		t.Fatalf("environment did not override redis config: %#v", loaded)
	}
	if loaded.AuthDisabled || loaded.LogtoIssuer != "https://env.example.com/oidc" {
		t.Fatalf("environment did not override auth config: %#v", loaded)
	}
}

func TestResetMarkerForcesSetupState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gravitylink.json")
	persisted := Config{
		AppEnv: "production", ConfigFile: path, AuthMode: "local",
		InstallationComplete: true, MySQLDSN: "user:pass@tcp(mysql:3306)/gravitylink",
		RedisAddr: "redis:6379",
	}
	if err := SaveFile(path, persisted); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := MarkReset(path); err != nil {
		t.Fatalf("mark reset: %v", err)
	}
	t.Setenv("CONFIG_FILE", path)

	loaded := Load()
	if !loaded.ResetPending || loaded.InstallationComplete {
		t.Fatalf("reset marker did not force setup state: %#v", loaded)
	}
	if err := ClearResetMarker(path); err != nil {
		t.Fatalf("clear reset marker: %v", err)
	}
	if _, err := os.Stat(path + ".reset"); !os.IsNotExist(err) {
		t.Fatalf("reset marker still exists: %v", err)
	}
}
