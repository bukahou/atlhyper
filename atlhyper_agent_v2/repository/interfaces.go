// Package repository 定义数据访问接口
//
// Repository 层负责数据访问，封装 SDK 调用并转换为 model_v2 类型。
// 上层 (Service) 只依赖 Repository 接口，不感知 K8s 原生类型。
//
// 设计原则:
//   - 每种 K8s 资源对应一个 Repository
//   - Repository 接口使用 model_v2 类型 (不暴露 K8s 类型)
//   - 类型转换在 converter.go 中完成
//
// 架构位置:
//
//	Service
//	    ↓ 调用
//	Repository (本包) ← 数据访问
//	    ↓ 调用
//	SDK                ← K8s 客户端
package repository

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/model_v2"
)

// =============================================================================
// Pod 仓库
// =============================================================================

// PodRepository Pod 数据访问接口 (只读 + 日志)
type PodRepository interface {
	// List 列出 Pod
	// namespace 为空则列出所有命名空间
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Pod, error)

	// Get 获取单个 Pod
	Get(ctx context.Context, namespace, name string) (*model_v2.Pod, error)

	// GetLogs 获取 Pod 日志
	GetLogs(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error)
}

// =============================================================================
// Node 仓库
// =============================================================================

// NodeRepository Node 数据访问接口 (只读)
type NodeRepository interface {
	// List 列出所有 Node
	List(ctx context.Context, opts model.ListOptions) ([]model_v2.Node, error)

	// Get 获取单个 Node
	Get(ctx context.Context, name string) (*model_v2.Node, error)
}

// =============================================================================
// Deployment 仓库
// =============================================================================

// DeploymentRepository Deployment 数据访问接口 (只读)
type DeploymentRepository interface {
	// List 列出 Deployment
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Deployment, error)

	// Get 获取单个 Deployment
	Get(ctx context.Context, namespace, name string) (*model_v2.Deployment, error)
}

// =============================================================================
// StatefulSet 仓库
// =============================================================================

// StatefulSetRepository StatefulSet 数据访问接口
type StatefulSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.StatefulSet, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.StatefulSet, error)
}

// =============================================================================
// DaemonSet 仓库
// =============================================================================

// DaemonSetRepository DaemonSet 数据访问接口
type DaemonSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.DaemonSet, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.DaemonSet, error)
}

// =============================================================================
// ReplicaSet 仓库
// =============================================================================

// ReplicaSetRepository ReplicaSet 数据访问接口
type ReplicaSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.ReplicaSet, error)
}

// =============================================================================
// Service 仓库
// =============================================================================

// ServiceRepository Service 数据访问接口
type ServiceRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Service, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Service, error)
}

// =============================================================================
// Ingress 仓库
// =============================================================================

// IngressRepository Ingress 数据访问接口
type IngressRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Ingress, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Ingress, error)
}

// =============================================================================
// ConfigMap 仓库
// =============================================================================

// ConfigMapRepository ConfigMap 数据访问接口
type ConfigMapRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.ConfigMap, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.ConfigMap, error)
}

// =============================================================================
// Secret 仓库
// =============================================================================

// SecretRepository Secret 数据访问接口
type SecretRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Secret, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Secret, error)
}

// =============================================================================
// Namespace 仓库
// =============================================================================

// NamespaceRepository Namespace 数据访问接口
type NamespaceRepository interface {
	List(ctx context.Context, opts model.ListOptions) ([]model_v2.Namespace, error)
	Get(ctx context.Context, name string) (*model_v2.Namespace, error)
}

// =============================================================================
// Event 仓库
// =============================================================================

// EventRepository Event 数据访问接口
type EventRepository interface {
	// List 列出 Event
	// 常用于查询某资源相关的事件
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Event, error)
}

// =============================================================================
// Job 仓库
// =============================================================================

// JobRepository Job 数据访问接口
type JobRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Job, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Job, error)
}

// =============================================================================
// CronJob 仓库
// =============================================================================

// CronJobRepository CronJob 数据访问接口
type CronJobRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.CronJob, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.CronJob, error)
}

// =============================================================================
// PV/PVC 仓库
// =============================================================================

// PersistentVolumeRepository PV 数据访问接口
type PersistentVolumeRepository interface {
	List(ctx context.Context, opts model.ListOptions) ([]model_v2.PersistentVolume, error)
}

// PersistentVolumeClaimRepository PVC 数据访问接口
type PersistentVolumeClaimRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.PersistentVolumeClaim, error)
}

// =============================================================================
// ResourceQuota 仓库
// =============================================================================

// ResourceQuotaRepository ResourceQuota 数据访问接口
type ResourceQuotaRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.ResourceQuota, error)
}

// =============================================================================
// LimitRange 仓库
// =============================================================================

// LimitRangeRepository LimitRange 数据访问接口
type LimitRangeRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.LimitRange, error)
}

// =============================================================================
// NetworkPolicy 仓库
// =============================================================================

// NetworkPolicyRepository NetworkPolicy 数据访问接口
type NetworkPolicyRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.NetworkPolicy, error)
}

// =============================================================================
// ServiceAccount 仓库
// =============================================================================

// ServiceAccountRepository ServiceAccount 数据访问接口
type ServiceAccountRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.ServiceAccount, error)
}

// =============================================================================
// SLO 仓库
// =============================================================================

// SLORepository SLO 数据仓库接口
//
// 负责从 Ingress Controller 采集 SLO 指标数据。
// 内部调用 sdk.IngressClient 采集原始指标，进行增量计算后返回。
//
// 与其他 Repository 一样，被 SnapshotService 注入和调用。
type SLORepository interface {
	// Collect 采集 SLO 指标数据
	//
	// 从 Ingress Controller 采集指标，计算增量，返回处理后的数据。
	// 如果 Ingress Controller 不可用或未发现，返回 nil 和 error。
	Collect(ctx context.Context) (*model_v2.SLOSnapshot, error)

	// CollectRoutes 采集 IngressRoute 配置
	//
	// 返回 Traefik service 名称到域名/路径的映射信息。
	CollectRoutes(ctx context.Context) ([]model_v2.IngressRouteInfo, error)
}

// =============================================================================
// 通用仓库
// =============================================================================

// GenericRepository 通用操作接口 (所有写操作 + 动态查询)
type GenericRepository interface {
	// =========================================================================
	// 删除操作
	// =========================================================================

	// DeletePod 删除 Pod
	DeletePod(ctx context.Context, namespace, name string, opts model.DeleteOptions) error

	// Delete 删除任意资源
	Delete(ctx context.Context, kind, namespace, name string, opts model.DeleteOptions) error

	// =========================================================================
	// Deployment 操作
	// =========================================================================

	// ScaleDeployment 扩缩容 Deployment
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error

	// RestartDeployment 重启 Deployment
	RestartDeployment(ctx context.Context, namespace, name string) error

	// UpdateDeploymentImage 更新容器镜像
	// container 为空时更新第一个容器
	UpdateDeploymentImage(ctx context.Context, namespace, name, container, image string) error

	// =========================================================================
	// Node 操作
	// =========================================================================

	// CordonNode 封锁节点 (设置 Unschedulable=true)
	CordonNode(ctx context.Context, name string) error

	// UncordonNode 解封节点 (设置 Unschedulable=false)
	UncordonNode(ctx context.Context, name string) error

	// =========================================================================
	// 配置数据获取
	// =========================================================================

	// GetConfigMapData 获取 ConfigMap 数据内容
	// 返回 key->value 映射
	GetConfigMapData(ctx context.Context, namespace, name string) (map[string]string, error)

	// GetSecretData 获取 Secret 数据内容
	// 返回 key->value 映射（base64 解码后）
	GetSecretData(ctx context.Context, namespace, name string) (map[string]string, error)

	// =========================================================================
	// 动态查询
	// =========================================================================

	// Execute 执行动态查询 (仅 GET)
	Execute(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error)
}
