// Package sdk 封装 K8s 客户端
//
// interfaces.go - K8sClient 接口定义
//
// SDK 层是 Agent 与 K8s API Server 交互的唯一入口。
// 上层代码只依赖 K8sClient 接口，完全隔离 client-go 细节。
//
// 设计原则:
//   - 接口返回原生 K8s 类型 (corev1.Pod 等)，由 Repository 层转换
//   - 所有方法接受 context，支持超时和取消
//   - 错误直接透传，不做额外包装
//
// 架构位置:
//
//	Repository
//	    ↓ 调用
//	SDK (本包) ← K8s 客户端封装
//	    ↓ 使用
//	client-go  ← K8s 官方库
//	    ↓
//	K8s API Server
package sdk

import (
	"context"

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

	// ListPods 列出 Pod
	// namespace 为空则列出所有命名空间
	ListPods(ctx context.Context, namespace string, opts ListOptions) ([]corev1.Pod, error)

	// GetPod 获取单个 Pod
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)

	// DeletePod 删除 Pod
	DeletePod(ctx context.Context, namespace, name string, opts DeleteOptions) error

	// GetPodLogs 获取 Pod 日志
	GetPodLogs(ctx context.Context, namespace, name string, opts LogOptions) (string, error)

	// =========================================================================
	// Node 操作
	// =========================================================================

	// ListNodes 列出所有 Node
	ListNodes(ctx context.Context, opts ListOptions) ([]corev1.Node, error)

	// GetNode 获取单个 Node
	GetNode(ctx context.Context, name string) (*corev1.Node, error)

	// ListNodeMetrics 获取所有 Node 的资源使用量
	// 需要集群安装 metrics-server，未安装时返回空 map
	ListNodeMetrics(ctx context.Context) (map[string]NodeMetrics, error)

	// ListPodMetrics 获取所有 Pod 的资源使用量
	// 返回 map[namespace/name]PodMetrics
	// 需要集群安装 metrics-server，未安装时返回空 map
	ListPodMetrics(ctx context.Context) (map[string]PodMetrics, error)

	// CordonNode 封锁节点 (设置 Unschedulable=true)
	CordonNode(ctx context.Context, name string) error

	// UncordonNode 解封节点 (设置 Unschedulable=false)
	UncordonNode(ctx context.Context, name string) error

	// =========================================================================
	// Deployment 操作
	// =========================================================================

	// ListDeployments 列出 Deployment
	ListDeployments(ctx context.Context, namespace string, opts ListOptions) ([]appsv1.Deployment, error)

	// GetDeployment 获取单个 Deployment
	GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error)

	// UpdateDeploymentScale 更新 Deployment 副本数
	UpdateDeploymentScale(ctx context.Context, namespace, name string, replicas int32) error

	// RestartDeployment 重启 Deployment
	// 通过更新 annotation 触发滚动重启
	RestartDeployment(ctx context.Context, namespace, name string) error

	// UpdateDeploymentImage 更新 Deployment 容器镜像
	// container 为空时更新第一个容器
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

	// Delete 删除资源
	Delete(ctx context.Context, gvk GroupVersionKind, namespace, name string, opts DeleteOptions) error

	// Dynamic 执行动态 API 查询 (仅 GET)
	Dynamic(ctx context.Context, req DynamicRequest) (*DynamicResponse, error)
}
