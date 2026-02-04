// Package config 配置管理
package config

import "time"

// Config 全局配置
type Config struct {
	NodeName string // 节点名称
	Hostname string // 主机名

	Log     LogConfig     // 日志配置
	Paths   PathsConfig   // 路径配置
	Collect CollectConfig // 采集配置
	Push    PushConfig    // 推送配置
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string // 日志级别: debug / info / warn / error
	Format string // 日志格式: text / json
}

// PathsConfig 路径配置
// 用于容器化部署时映射宿主机路径
type PathsConfig struct {
	ProcRoot string // /proc 路径，默认 /proc，容器中使用 /host_proc
	SysRoot  string // /sys 路径，默认 /sys，容器中使用 /host_sys
	HostRoot string // 宿主机根路径，默认 /，容器中使用 /host_root
}

// CollectConfig 采集配置
type CollectConfig struct {
	TopProcesses  int           // Top N 进程数量
	CPUInterval   time.Duration // CPU 采样间隔
	ProcInterval  time.Duration // 进程采样间隔
	CollectInterval time.Duration // 总采集间隔
}

// PushConfig 推送配置
type PushConfig struct {
	AgentAddr  string        // Agent 地址
	Timeout    time.Duration // 请求超时
	RetryCount int           // 重试次数
	RetryDelay time.Duration // 重试间隔
}
