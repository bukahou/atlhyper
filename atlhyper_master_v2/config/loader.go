// atlhyper_master_v2/config/loader.go
// 配置加载：变量名、默认值、必填见 types.go 的字段标签
package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"

	"AtlHyper/common/logger"
)

var log = logger.Module("Config")

// oidcCookieSecretMinBytes 状态 cookie 签名密钥的最短长度
const oidcCookieSecretMinBytes = 32

// Load 从环境变量读取配置。
// 格式写错、必填缺失都返回错误并指出变量名，⛔ 不静默用默认值；为空的变量按未设置处理（取默认值）。
func Load() (AppConfig, error) {
	cfg, err := env.ParseAs[AppConfig]()
	if err != nil {
		return AppConfig{}, withEnvKeys(err)
	}
	if err := cfg.validate(); err != nil {
		return AppConfig{}, err
	}
	return cfg, nil
}

// LoadConfig 加载配置到 GlobalConfig；配置无效时记日志并退出（启动入口用）
func LoadConfig() {
	cfg, err := Load()
	if err != nil {
		log.Error("配置无效", "err", err)
		os.Exit(1)
	}
	GlobalConfig = cfg
	log.Info("Master 配置加载完成",
		"gatewayPort", cfg.Server.GatewayPort,
		"agentSDKPort", cfg.Server.AgentSDKPort,
		"dbType", cfg.Database.Type,
		"admin", cfg.Admin.Username,
		"oidc", cfg.OIDC.Enabled())
}

// validate 跨字段的校验（单个字段的必填已由标签声明）
func (c AppConfig) validate() error {
	if !c.OIDC.Enabled() {
		return nil
	}
	// 启用 OIDC 时凭证类配置必填：忘配时启动即失败，优于「默默登不进去」
	var errs []error
	for key, value := range map[string]string{
		"MASTER_OIDC_CLIENT_ID":     c.OIDC.ClientID,
		"MASTER_OIDC_CLIENT_SECRET": c.OIDC.ClientSecret,
		"MASTER_OIDC_REDIRECT_URL":  c.OIDC.RedirectURL,
		"MASTER_OIDC_COOKIE_SECRET": c.OIDC.CookieSecret,
	} {
		if value == "" {
			errs = append(errs, fmt.Errorf("%s: 启用 OIDC（设置了 MASTER_OIDC_ISSUER）时必填", key))
		}
	}
	if c.OIDC.CookieSecret != "" && len(c.OIDC.CookieSecret) < oidcCookieSecretMinBytes {
		errs = append(errs, fmt.Errorf("MASTER_OIDC_COOKIE_SECRET: 过短（需 ≥ %d 字节）", oidcCookieSecretMinBytes))
	}
	return errors.Join(errs...)
}

// withEnvKeys 解析错误只带 Go 字段名，换成运维认得的环境变量名
func withEnvKeys(err error) error {
	var agg env.AggregateError
	if !errors.As(err, &agg) {
		return err
	}
	keys := map[string][]string{}
	envKeysByField(reflect.TypeOf(AppConfig{}), keys)
	out := make([]error, 0, len(agg.Errors))
	for _, e := range agg.Errors {
		var pe env.ParseError
		if errors.As(e, &pe) && len(keys[pe.Name]) > 0 {
			out = append(out, fmt.Errorf("%s: 格式不对: %w", strings.Join(keys[pe.Name], " / "), pe.Err))
			continue
		}
		out = append(out, e)
	}
	return errors.Join(out...)
}

func envKeysByField(t reflect.Type, out map[string][]string) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Tag.Get("env") == "" && f.Type.PkgPath() == t.PkgPath() {
			envKeysByField(f.Type, out)
			continue
		}
		if key, _, _ := strings.Cut(f.Tag.Get("env"), ","); key != "" {
			out[f.Name] = append(out[f.Name], key)
		}
	}
}

// CSVList 逗号分隔的列表：去掉每项两侧空白、丢掉空项
type CSVList []string

// UnmarshalText 实现 encoding.TextUnmarshaler（env 解析时调用）
func (l *CSVList) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), ",")
	items := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			items = append(items, p)
		}
	}
	*l = items
	return nil
}
