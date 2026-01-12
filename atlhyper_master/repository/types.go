// atlhyper_master/repository/types.go
// 仓库层类型定义
package repository

import (
	"time"

	"AtlHyper/model/collect"
	"AtlHyper/model/k8s"
	"AtlHyper/model/transport"
)

// ============================================================
// K8s 资源类型别名（内存仓库使用）
// ============================================================

type Pod = k8s.Pod
type Node = k8s.Node
type Service = k8s.Service
type Namespace = k8s.Namespace
type Ingress = k8s.Ingress
type Deployment = k8s.Deployment
type ConfigMap = k8s.ConfigMap

type LogEvent = transport.LogEvent
type NodeMetricsSnapshot = collect.NodeMetricsSnapshot

// ============================================================
// SQL 仓库类型（持久化数据）
// ============================================================

// AuditLog 审计日志
type AuditLog struct {
	ID        int
	UserID    int
	Username  string
	Role      int
	Action    string
	Success   bool
	IP        string
	Method    string
	Status    int
	Timestamp string
}

// SlackConfig Slack 配置
type SlackConfig struct {
	ID          int64
	Enable      bool
	Webhook     string
	IntervalSec int64
	UpdatedAt   time.Time
}

// MailConfig 邮件配置
type MailConfig struct {
	ID          int64
	Enable      bool
	SMTPHost    string
	SMTPPort    string
	Username    string
	Password    string
	FromAddr    string
	ToAddrs     string // 逗号分隔
	IntervalSec int64
	UpdatedAt   time.Time
}

// NodeMetricsFlat 节点指标（扁平化，用于持久化）
type NodeMetricsFlat struct {
	ID              int64
	NodeName        string
	Timestamp       string
	CPUUsage        float64
	CPUCores        int
	CPULoad1        float64
	CPULoad5        float64
	CPULoad15       float64
	MemoryTotal     int64
	MemoryUsed      int64
	MemoryAvailable int64
	MemoryUsage     float64
	TempCPU         int64
	TempGPU         int64
	TempNVME        int64
	DiskTotal       int64
	DiskUsed        int64
	DiskFree        int64
	DiskUsage       float64
	NetLoRxKBps     float64
	NetLoTxKBps     float64
	NetEth0RxKBps   float64
	NetEth0TxKBps   float64
}

// TopProcess 进程信息
type TopProcess struct {
	NodeName   string
	Timestamp  string
	PID        int
	User       string
	Command    string
	CPUPercent float64
	MemoryMB   float64
}
