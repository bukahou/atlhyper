package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// clearEnv 清掉本进程里与主控配置有关的变量（开发机的终端会预先加载这些变量）
func clearEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "MASTER_") || strings.HasPrefix(key, "GITHUB_") {
			t.Setenv(key, "")
			os.Unsetenv(key)
		}
	}
}

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("MASTER_JWT_SECRET", "jwt-secret-placeholder-not-real")
	t.Setenv("MASTER_ADMIN_USERNAME", "admin-placeholder")
	t.Setenv("MASTER_ADMIN_PASSWORD", "password-placeholder")
}

func mustLoad(t *testing.T) AppConfig {
	t.Helper()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// 只给必填项时，每个配置项取到的默认值（变量名与默认值改造前后都不变）
func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	c := mustLoad(t)

	checks := []struct {
		name      string
		got, want any
	}{
		{"Log.Level", c.Log.Level, "info"},
		{"Log.Format", c.Log.Format, "text"},
		{"Server.GatewayPort", c.Server.GatewayPort, 8080},
		{"Server.AgentSDKPort", c.Server.AgentSDKPort, 8081},
		{"Server.PublicURL", c.Server.PublicURL, ""},
		{"Server.Timezone", c.Server.Timezone, "UTC"},
		{"Server.ClientIPHeader", c.Server.ClientIPHeader, ""},
		{"Server.MaxRequestBodyBytes", c.Server.MaxRequestBodyBytes(), int64(1024 << 10)},
		{"Server.RateLimitPerMinute", c.Server.RateLimitPerMinute, 3000},
		{"Server.LoginRateLimitPerMinute", c.Server.LoginRateLimitPerMinute, 20},
		{"AgentSDK.AuthMode", c.AgentSDK.AuthMode, "warn"},
		{"AgentSDK.MaxSnapshotMB", c.AgentSDK.MaxSnapshotMB, 256},
		{"DataHub.EventRetention", c.DataHub.EventRetention, 30 * time.Minute},
		{"DataHub.HeartbeatExpire", c.DataHub.HeartbeatExpire, 45 * time.Second},
		{"DataHub.SnapshotRetention", c.DataHub.SnapshotRetention, 15 * time.Minute},
		{"Database.Type", c.Database.Type, "sqlite"},
		{"Database.Path", c.Database.Path, "atlhyper_master_v2/database/sqlite/data/master.db"},
		{"Database.CommandHistoryRetention", c.Database.CommandHistoryRetention, 2160 * time.Hour},
		{"Database.AuditRetention", c.Database.AuditRetention, 4320 * time.Hour},
		{"Database.AIOpsIncidentRetention", c.Database.AIOpsIncidentRetention, 4320 * time.Hour},
		{"Database.AIReportRetention", c.Database.AIReportRetention, 4320 * time.Hour},
		{"Event.RetentionDays", c.Event.RetentionDays, 30},
		{"Event.MaxCount", c.Event.MaxCount, 100000},
		{"Event.CleanupInterval", c.Event.CleanupInterval, time.Hour},
		{"EventAlert.CheckInterval", c.EventAlert.CheckInterval, 30 * time.Second},
		{"Timeout.CommandPoll", c.Timeout.CommandPoll, 60 * time.Second},
		{"JWT.SecretKey", c.JWT.SecretKey, "jwt-secret-placeholder-not-real"},
		{"JWT.TokenExpiry", c.JWT.TokenExpiry, 24 * time.Hour},
		{"Admin.Username", c.Admin.Username, "admin-placeholder"},
		{"Admin.Password", c.Admin.Password, "password-placeholder"},
		{"Admin.DisplayName", c.Admin.DisplayName, ""},
		{"AI.ToolTimeout", c.AI.ToolTimeout, 30 * time.Second},
		{"AI.BedrockRegion", c.AI.BedrockRegion, "ap-northeast-1"},
		{"AI.GitHubPrivateRepos", len(c.AI.GitHubPrivateRepos), 0},
		{"AI.Seed", c.AI.Seed, AISeed{}},
		{"GitHub.AppID", c.GitHub.AppID, int64(0)},
		{"GitHub.AppSlug", c.GitHub.AppSlug, ""},
		{"GitHub.PrivateKeyPath", c.GitHub.PrivateKeyPath, ""},
		{"GitHub.CallbackURL", c.GitHub.CallbackURL, "http://localhost:3000/auth/github/callback"},
		{"OIDC.Issuer", c.OIDC.Issuer, ""},
		{"OIDC.CookieSecure", c.OIDC.CookieSecure, true},
		{"OIDC.DisplayName", c.OIDC.DisplayName, "SSO"},
	}
	for _, ck := range checks {
		if !reflect.DeepEqual(ck.got, ck.want) {
			t.Errorf("%s = %#v, want %#v", ck.name, ck.got, ck.want)
		}
	}
}

// 每个变量名都接上了：给一个非默认值，读出来就是它（防字段标签写错名字）
func TestLoad_EnvOverrides(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	env := map[string]string{
		"MASTER_LOG_LEVEL": "debug", "MASTER_LOG_FORMAT": "json",
		"MASTER_GATEWAY_PORT": "9090", "MASTER_AGENTSDK_PORT": "9091",
		"MASTER_PUBLIC_URL": "https://atlhyper.example.com", "MASTER_TIMEZONE": "Asia/Tokyo",
		"MASTER_GATEWAY_CLIENT_IP_HEADER": "X-Real-IP", "MASTER_GATEWAY_MAX_BODY_KB": "64",
		"MASTER_GATEWAY_RATE_LIMIT_PER_MINUTE": "100", "MASTER_GATEWAY_LOGIN_RATE_LIMIT_PER_MINUTE": "5",
		"MASTER_AGENT_AUTH_MODE": "enforce", "MASTER_AGENTSDK_MAX_SNAPSHOT_MB": "64",
		"MASTER_DATAHUB_EVENT_RETENTION": "5m", "MASTER_DATAHUB_HEARTBEAT_EXPIRE": "1m", "MASTER_DATAHUB_SNAPSHOT_RETENTION": "2m",
		"MASTER_DB_TYPE": "sqlite", "MASTER_DB_PATH": "/data/master.db",
		"MASTER_COMMAND_HISTORY_RETENTION": "1h", "MASTER_AUDIT_RETENTION": "2h",
		"MASTER_AIOPS_INCIDENT_RETENTION": "3h", "MASTER_AI_REPORT_RETENTION": "4h",
		"MASTER_EVENT_RETENTION_DAYS": "7", "MASTER_EVENT_MAX_COUNT": "500", "MASTER_EVENT_CLEANUP_INTERVAL": "10m",
		"MASTER_EVENT_ALERT_INTERVAL": "11s", "MASTER_TIMEOUT_COMMAND_POLL": "12s",
		"MASTER_JWT_TOKEN_EXPIRY": "13h", "MASTER_ADMIN_DISPLAY_NAME": "Ops",
		"MASTER_AI_TOOL_TIMEOUT": "14s", "MASTER_AI_BEDROCK_REGION": "us-east-1",
		"MASTER_AI_GITHUB_PRIVATE_REPOS": "example-org/a, example-org/b",
		"MASTER_AI_SEED_PROVIDER":        "ollama", "MASTER_AI_SEED_NAME": "local", "MASTER_AI_SEED_API_KEY": "key-placeholder",
		"MASTER_AI_SEED_MODEL": "m", "MASTER_AI_SEED_BASE_URL": "http://ollama.example.com",
		"GITHUB_APP_ID": "42", "GITHUB_APP_SLUG": "slug", "GITHUB_PRIVATE_KEY_PATH": "/k.pem",
		"GITHUB_CALLBACK_URL": "https://example.com/cb",
		"MASTER_OIDC_ISSUER":  "https://id.example.com", "MASTER_OIDC_CLIENT_ID": "cid",
		"MASTER_OIDC_CLIENT_SECRET": "secret-placeholder", "MASTER_OIDC_REDIRECT_URL": "https://example.com/oidc",
		"MASTER_OIDC_COOKIE_SECRET": "cookie-secret-placeholder-32-bytes-long", "MASTER_OIDC_COOKIE_SECURE": "false",
		"MASTER_OIDC_DISPLAY_NAME": "Akasha",
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
	c := mustLoad(t)
	want := map[string][2]any{
		"MASTER_LOG_LEVEL":                           {c.Log.Level, "debug"},
		"MASTER_LOG_FORMAT":                          {c.Log.Format, "json"},
		"MASTER_GATEWAY_PORT":                        {c.Server.GatewayPort, 9090},
		"MASTER_AGENTSDK_PORT":                       {c.Server.AgentSDKPort, 9091},
		"MASTER_PUBLIC_URL":                          {c.Server.PublicURL, "https://atlhyper.example.com"},
		"MASTER_TIMEZONE":                            {c.Server.Timezone, "Asia/Tokyo"},
		"MASTER_GATEWAY_CLIENT_IP_HEADER":            {c.Server.ClientIPHeader, "X-Real-IP"},
		"MASTER_GATEWAY_MAX_BODY_KB":                 {c.Server.MaxRequestBodyBytes(), int64(64 << 10)},
		"MASTER_GATEWAY_RATE_LIMIT_PER_MINUTE":       {c.Server.RateLimitPerMinute, 100},
		"MASTER_GATEWAY_LOGIN_RATE_LIMIT_PER_MINUTE": {c.Server.LoginRateLimitPerMinute, 5},
		"MASTER_AGENT_AUTH_MODE":                     {c.AgentSDK.AuthMode, "enforce"},
		"MASTER_AGENTSDK_MAX_SNAPSHOT_MB":            {c.AgentSDK.MaxSnapshotMB, 64},
		"MASTER_DATAHUB_EVENT_RETENTION":             {c.DataHub.EventRetention, 5 * time.Minute},
		"MASTER_DATAHUB_HEARTBEAT_EXPIRE":            {c.DataHub.HeartbeatExpire, time.Minute},
		"MASTER_DATAHUB_SNAPSHOT_RETENTION":          {c.DataHub.SnapshotRetention, 2 * time.Minute},
		"MASTER_DB_PATH":                             {c.Database.Path, "/data/master.db"},
		"MASTER_COMMAND_HISTORY_RETENTION":           {c.Database.CommandHistoryRetention, time.Hour},
		"MASTER_AUDIT_RETENTION":                     {c.Database.AuditRetention, 2 * time.Hour},
		"MASTER_AIOPS_INCIDENT_RETENTION":            {c.Database.AIOpsIncidentRetention, 3 * time.Hour},
		"MASTER_AI_REPORT_RETENTION":                 {c.Database.AIReportRetention, 4 * time.Hour},
		"MASTER_EVENT_RETENTION_DAYS":                {c.Event.RetentionDays, 7},
		"MASTER_EVENT_MAX_COUNT":                     {c.Event.MaxCount, 500},
		"MASTER_EVENT_CLEANUP_INTERVAL":              {c.Event.CleanupInterval, 10 * time.Minute},
		"MASTER_EVENT_ALERT_INTERVAL":                {c.EventAlert.CheckInterval, 11 * time.Second},
		"MASTER_TIMEOUT_COMMAND_POLL":                {c.Timeout.CommandPoll, 12 * time.Second},
		"MASTER_JWT_TOKEN_EXPIRY":                    {c.JWT.TokenExpiry, 13 * time.Hour},
		"MASTER_ADMIN_DISPLAY_NAME":                  {c.Admin.DisplayName, "Ops"},
		"MASTER_AI_TOOL_TIMEOUT":                     {c.AI.ToolTimeout, 14 * time.Second},
		"MASTER_AI_BEDROCK_REGION":                   {c.AI.BedrockRegion, "us-east-1"},
		"MASTER_AI_GITHUB_PRIVATE_REPOS":             {[]string(c.AI.GitHubPrivateRepos), []string{"example-org/a", "example-org/b"}},
		"MASTER_AI_SEED_*":                           {c.AI.Seed, AISeed{Provider: "ollama", Name: "local", APIKey: "key-placeholder", Model: "m", BaseURL: "http://ollama.example.com"}},
		"GITHUB_APP_ID":                              {c.GitHub.AppID, int64(42)},
		"GITHUB_APP_SLUG":                            {c.GitHub.AppSlug, "slug"},
		"GITHUB_PRIVATE_KEY_PATH":                    {c.GitHub.PrivateKeyPath, "/k.pem"},
		"GITHUB_CALLBACK_URL":                        {c.GitHub.CallbackURL, "https://example.com/cb"},
		"MASTER_OIDC_*": {c.OIDC, OIDCConfig{Issuer: "https://id.example.com", ClientID: "cid", ClientSecret: "secret-placeholder",
			RedirectURL: "https://example.com/oidc", CookieSecret: "cookie-secret-placeholder-32-bytes-long", CookieSecure: false, DisplayName: "Akasha"}},
	}
	for name, pair := range want {
		if !reflect.DeepEqual(pair[0], pair[1]) {
			t.Errorf("%s: got %#v, want %#v", name, pair[0], pair[1])
		}
	}
}

// 格式写错 / 必填缺失：返回错误并指出变量名，⛔ 不静默用默认值
func TestLoad_InvalidValuesFail(t *testing.T) {
	cases := map[string]map[string]string{
		"MASTER_GATEWAY_PORT":         {"MASTER_GATEWAY_PORT": "80a"},
		"MASTER_TIMEOUT_COMMAND_POLL": {"MASTER_TIMEOUT_COMMAND_POLL": "60"},
		"MASTER_OIDC_COOKIE_SECURE":   {"MASTER_OIDC_COOKIE_SECURE": "ture"},
		"MASTER_JWT_SECRET":           {"MASTER_JWT_SECRET": ""},
		"MASTER_ADMIN_PASSWORD":       {"MASTER_ADMIN_PASSWORD": ""},
	}
	for key, env := range cases {
		t.Run(key, func(t *testing.T) {
			clearEnv(t)
			setRequired(t)
			for k, v := range env {
				t.Setenv(k, v)
			}
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("应返回指出 %s 的错误，实际 %v", key, err)
			}
		})
	}
}

// 启用 OIDC 时凭证类配置必填，Cookie 密钥至少 32 字节
func TestLoad_OIDCRequiresCredentials(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	t.Setenv("MASTER_OIDC_ISSUER", "https://id.example.com")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "MASTER_OIDC_CLIENT_ID") {
		t.Fatalf("启用 OIDC 而缺 client_id 应失败，实际 %v", err)
	}
	t.Setenv("MASTER_OIDC_CLIENT_ID", "cid")
	t.Setenv("MASTER_OIDC_CLIENT_SECRET", "secret-placeholder")
	t.Setenv("MASTER_OIDC_REDIRECT_URL", "https://example.com/oidc")
	t.Setenv("MASTER_OIDC_COOKIE_SECRET", "too-short")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "MASTER_OIDC_COOKIE_SECRET") {
		t.Fatalf("Cookie 密钥过短应失败，实际 %v", err)
	}
}
