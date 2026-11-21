// atlhyper_agent_v2/config/loader.go
// 配置加载：变量名与默认值见 types.go 的字段标签
package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Load 从环境变量读取配置。
// 格式写错返回错误并指出变量名，⛔ 不静默用默认值；为空的变量按未设置处理（取默认值）。
func Load() (AppConfig, error) {
	cfg, err := env.ParseAs[AppConfig]()
	if err != nil {
		return AppConfig{}, withEnvKeys(err)
	}
	// 从文件或 Secret 注入的令牌常带换行，而带换行的值放不进请求头
	cfg.Master.Token = strings.TrimSpace(cfg.Master.Token)
	if cfg.Master.Token == "" {
		// notEmpty 只拦空串，只有空白的值去掉空白后同样等于没配
		return AppConfig{}, errors.New("AGENT_MASTER_TOKEN 只有空白：请填入在「集群管理」页签发的集群令牌")
	}
	if cfg.Kubernetes.KubeConfig == "" {
		cfg.Kubernetes.KubeConfig = os.Getenv("KUBECONFIG")
	}
	return cfg, nil
}

// LoadConfig 加载配置到 GlobalConfig；配置无效时退出（启动入口用）
func LoadConfig() {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("[config] 配置无效: %v", err)
	}
	GlobalConfig = cfg

	kube := cfg.Kubernetes.KubeConfig
	if kube == "" {
		kube = "in-cluster"
	}
	// 令牌是凭证：日志只说明有没有配置，不打印内容
	log.Printf("[config] Agent 配置加载完成: ClusterID=%s, MasterURL=%s, MasterToken=%s, Kube=%s, CH=%s/%s",
		cfg.Agent.ClusterID, cfg.Master.URL, describeSecretPresence(cfg.Master.Token), kube,
		RedactedEndpoint(cfg.ClickHouse.Endpoint), cfg.ClickHouse.Database)
}

// RedactedEndpoint 地址里可能带账号密码（ClickHouse 只能这样传凭据）：解析后把密码换掉再打日志。
// 解析不了的地址不打印原文
func RedactedEndpoint(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<地址格式无法解析>"
	}
	return u.Redacted()
}

// describeSecretPresence 凭证类配置在日志里的写法：只有「已配置 / 未配置」
func describeSecretPresence(value string) string {
	if value == "" {
		return "未配置"
	}
	return "已配置"
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
