// atlhyper_metrics/config/types.go
// 配置结构体定义
package config

import "time"

// PushConfig 主动上报相关配置
type PushConfig struct {
	Enable   bool          // 是否启用主动上报
	URL      string        // Agent 接收端 URL
	Token    string        // 可选：Bearer Token
	Interval time.Duration // 上报间隔
	Timeout  time.Duration // HTTP 超时
}

// CollectConfig 指标采集相关配置
type CollectConfig struct {
	NodeName string // 节点名称（用于标识上报来源）
	ProcRoot string // /proc 路径（容器场景下可能挂载宿主机路径）
	SysRoot  string // /sys 路径（容器场景下可能挂载宿主机路径）
}

// Config 模块总配置
type Config struct {
	Push    PushConfig
	Collect CollectConfig
}

// C 全局配置实例
var C Config
