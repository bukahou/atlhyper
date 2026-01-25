// atlhyper_agent/config/types.go
// 配置结构体定义
package config

import "time"

// DiagnosisConfig 诊断系统配置
type DiagnosisConfig struct {
	CleanInterval            time.Duration // 清理器执行间隔
	RetentionRawDuration     time.Duration // 原始事件保留时间
	RetentionCleanedDuration time.Duration // 清理池保留时间
}

// KubernetesConfig Kubernetes 相关配置
type KubernetesConfig struct {
	Kubeconfig             string        // kubeconfig 文件路径（空则使用 InCluster）
	APIHealthCheckInterval time.Duration // K8s API 健康检查间隔
}

// ClusterConfig 集群标识配置
type ClusterConfig struct {
	ClusterID string // 集群唯一标识（空则自动获取 kube-system UID）
}

// PushConfig 推送配置
type PushConfig struct {
	MasterURL    string        // Master 服务地址
	PushInterval time.Duration // 推送间隔
	Timeout      time.Duration // 请求超时
}

// RestClientConfig REST 客户端配置
type RestClientConfig struct {
	BaseURL      string        // API 基础地址
	Timeout      time.Duration // 单次请求超时
	MaxRespBytes int64         // 响应体读取上限（字节）
	Gzip         bool          // 是否启用 gzip 压缩
}

// StoreConfig 内存存储配置
type StoreConfig struct {
	TTLMaxAge       time.Duration // 数据最大存活时间
	CleanupInterval time.Duration // 清理器执行间隔
}

// ServerConfig Agent 服务器配置
type ServerConfig struct {
	Port string
}

// AppConfig Agent 顶层配置结构体
type AppConfig struct {
	Diagnosis  DiagnosisConfig
	Kubernetes KubernetesConfig
	Cluster    ClusterConfig
	Push       PushConfig
	RestClient RestClientConfig
	Store      StoreConfig
	Server     ServerConfig
}

// GlobalConfig 全局配置实例
var GlobalConfig AppConfig
