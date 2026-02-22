// Package sdk 封装外部客户端
//
// interfaces.go - 统一接口定义
//
// SDK 层是 Agent 与外部系统交互的唯一入口。
// 上层代码只依赖本包接口，完全隔离底层实现细节。
//
// 客户端:
//   - K8sClient:        封装 client-go，主动拉取 K8s API Server
//   - ClickHouseClient: 封装 clickhouse-go，查询 ClickHouse 时序数据
//
// 架构位置:
//
//	Repository
//	    ↓ 调用
//	SDK (本包) ← 外部客户端封装
//	    ↓ 使用
//	client-go / clickhouse-go
//	    ↓
//	K8s API Server / ClickHouse
package sdk

import (
	"context"
	"database/sql"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// K8sClient K8s 客户端接口
//
// 封装所有 K8s API 操作。上层只依赖此接口，不直接使用 client-go。
// 接口按资源类型分组，每组包含 List/Get 等操作。
type K8sClient interface {
	// =========================================================================
	// Pod 操作
	// =========================================================================

	ListPods(ctx context.Context, namespace string, opts ListOptions) ([]corev1.Pod, error)
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string, opts DeleteOptions) error
	GetPodLogs(ctx context.Context, namespace, name string, opts LogOptions) (string, error)

	// =========================================================================
	// Node 操作
	// =========================================================================

	ListNodes(ctx context.Context, opts ListOptions) ([]corev1.Node, error)
	GetNode(ctx context.Context, name string) (*corev1.Node, error)
	ListNodeMetrics(ctx context.Context) (map[string]NodeMetrics, error)
	ListPodMetrics(ctx context.Context) (map[string]PodMetrics, error)
	CordonNode(ctx context.Context, name string) error
	UncordonNode(ctx context.Context, name string) error

	// =========================================================================
	// Deployment 操作
	// =========================================================================

	ListDeployments(ctx context.Context, namespace string, opts ListOptions) ([]appsv1.Deployment, error)
	GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error)
	UpdateDeploymentScale(ctx context.Context, namespace, name string, replicas int32) error
	RestartDeployment(ctx context.Context, namespace, name string) error
	UpdateDeploymentImage(ctx context.Context, namespace, name, container, image string) error

	// =========================================================================
	// StatefulSet 操作
	// =========================================================================

	ListStatefulSets(ctx context.Context, namespace string, opts ListOptions) ([]appsv1.StatefulSet, error)
	GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error)

	// =========================================================================
	// DaemonSet 操作
	// =========================================================================

	ListDaemonSets(ctx context.Context, namespace string, opts ListOptions) ([]appsv1.DaemonSet, error)
	GetDaemonSet(ctx context.Context, namespace, name string) (*appsv1.DaemonSet, error)

	// =========================================================================
	// ReplicaSet 操作
	// =========================================================================

	ListReplicaSets(ctx context.Context, namespace string, opts ListOptions) ([]appsv1.ReplicaSet, error)

	// =========================================================================
	// Service 操作
	// =========================================================================

	ListServices(ctx context.Context, namespace string, opts ListOptions) ([]corev1.Service, error)
	GetService(ctx context.Context, namespace, name string) (*corev1.Service, error)

	// =========================================================================
	// Ingress 操作
	// =========================================================================

	ListIngresses(ctx context.Context, namespace string, opts ListOptions) ([]networkingv1.Ingress, error)
	GetIngress(ctx context.Context, namespace, name string) (*networkingv1.Ingress, error)

	// =========================================================================
	// ConfigMap 操作
	// =========================================================================

	ListConfigMaps(ctx context.Context, namespace string, opts ListOptions) ([]corev1.ConfigMap, error)
	GetConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error)

	// =========================================================================
	// Secret 操作
	// =========================================================================

	ListSecrets(ctx context.Context, namespace string, opts ListOptions) ([]corev1.Secret, error)
	GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error)

	// =========================================================================
	// Namespace 操作
	// =========================================================================

	ListNamespaces(ctx context.Context, opts ListOptions) ([]corev1.Namespace, error)
	GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error)

	// =========================================================================
	// Event 操作
	// =========================================================================

	ListEvents(ctx context.Context, namespace string, opts ListOptions) ([]corev1.Event, error)

	// =========================================================================
	// Job 操作
	// =========================================================================

	ListJobs(ctx context.Context, namespace string, opts ListOptions) ([]batchv1.Job, error)
	GetJob(ctx context.Context, namespace, name string) (*batchv1.Job, error)

	// =========================================================================
	// CronJob 操作
	// =========================================================================

	ListCronJobs(ctx context.Context, namespace string, opts ListOptions) ([]batchv1.CronJob, error)
	GetCronJob(ctx context.Context, namespace, name string) (*batchv1.CronJob, error)

	// =========================================================================
	// PV/PVC 操作
	// =========================================================================

	ListPersistentVolumes(ctx context.Context, opts ListOptions) ([]corev1.PersistentVolume, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string, opts ListOptions) ([]corev1.PersistentVolumeClaim, error)

	// =========================================================================
	// ResourceQuota 操作
	// =========================================================================

	ListResourceQuotas(ctx context.Context, namespace string, opts ListOptions) ([]corev1.ResourceQuota, error)

	// =========================================================================
	// LimitRange 操作
	// =========================================================================

	ListLimitRanges(ctx context.Context, namespace string, opts ListOptions) ([]corev1.LimitRange, error)

	// =========================================================================
	// NetworkPolicy 操作
	// =========================================================================

	ListNetworkPolicies(ctx context.Context, namespace string, opts ListOptions) ([]networkingv1.NetworkPolicy, error)

	// =========================================================================
	// ServiceAccount 操作
	// =========================================================================

	ListServiceAccounts(ctx context.Context, namespace string, opts ListOptions) ([]corev1.ServiceAccount, error)

	// =========================================================================
	// 通用操作
	// =========================================================================

	Delete(ctx context.Context, gvk GroupVersionKind, namespace, name string, opts DeleteOptions) error
	Dynamic(ctx context.Context, req DynamicRequest) (*DynamicResponse, error)
}

// =============================================================================
// ClickHouse 客户端
// =============================================================================

// ClickHouseClient ClickHouse 查询客户端接口
//
// 封装 ClickHouse 数据库操作。用于查询 OTel 指标、日志、追踪等时序数据。
//
// 架构位置:
//
//	CH Repository
//	    ↓ 调用
//	SDK (ClickHouseClient)
//	    ↓ 使用
//	clickhouse-go
//	    ↓
//	ClickHouse Server
type ClickHouseClient interface {
	// Query 执行查询，返回多行结果
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// QueryRow 执行查询，返回单行结果
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row

	// Ping 检查连接健康
	Ping(ctx context.Context) error

	// Close 关闭连接
	Close() error
}
