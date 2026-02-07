// Package repository 定义数据访问接口
//
// 所有 Repository 接口统一在此文件定义，实现按职责分布在子包中:
//   - k8s/       K8s 资源仓库 (通过 K8s API Server 采集)
//   - metrics/   硬件指标仓库 (接收 atlhyper_metrics_v2 推送)
//   - slo/       SLO 指标仓库 (通过 Ingress Controller 采集)
//
// 上层 (Service) 只依赖本包接口，不感知具体实现。
package repository

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/model_v2"
)

// =============================================================================
// K8s 资源仓库 — 实现: repository/k8s/
// =============================================================================

// PodRepository Pod 数据访问接口
type PodRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Pod, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Pod, error)
	GetLogs(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error)
}

// NodeRepository Node 数据访问接口
type NodeRepository interface {
	List(ctx context.Context, opts model.ListOptions) ([]model_v2.Node, error)
	Get(ctx context.Context, name string) (*model_v2.Node, error)
}

// DeploymentRepository Deployment 数据访问接口
type DeploymentRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Deployment, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Deployment, error)
}

// StatefulSetRepository StatefulSet 数据访问接口
type StatefulSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.StatefulSet, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.StatefulSet, error)
}

// DaemonSetRepository DaemonSet 数据访问接口
type DaemonSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.DaemonSet, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.DaemonSet, error)
}

// ReplicaSetRepository ReplicaSet 数据访问接口
type ReplicaSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.ReplicaSet, error)
}

// ServiceRepository Service 数据访问接口
type ServiceRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Service, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Service, error)
}

// IngressRepository Ingress 数据访问接口
type IngressRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Ingress, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Ingress, error)
}

// ConfigMapRepository ConfigMap 数据访问接口
type ConfigMapRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.ConfigMap, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.ConfigMap, error)
}

// SecretRepository Secret 数据访问接口
type SecretRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Secret, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Secret, error)
}

// NamespaceRepository Namespace 数据访问接口
type NamespaceRepository interface {
	List(ctx context.Context, opts model.ListOptions) ([]model_v2.Namespace, error)
	Get(ctx context.Context, name string) (*model_v2.Namespace, error)
}

// EventRepository Event 数据访问接口
type EventRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Event, error)
}

// JobRepository Job 数据访问接口
type JobRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Job, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.Job, error)
}

// CronJobRepository CronJob 数据访问接口
type CronJobRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.CronJob, error)
	Get(ctx context.Context, namespace, name string) (*model_v2.CronJob, error)
}

// PersistentVolumeRepository PV 数据访问接口
type PersistentVolumeRepository interface {
	List(ctx context.Context, opts model.ListOptions) ([]model_v2.PersistentVolume, error)
}

// PersistentVolumeClaimRepository PVC 数据访问接口
type PersistentVolumeClaimRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.PersistentVolumeClaim, error)
}

// ResourceQuotaRepository ResourceQuota 数据访问接口
type ResourceQuotaRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.ResourceQuota, error)
}

// LimitRangeRepository LimitRange 数据访问接口
type LimitRangeRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.LimitRange, error)
}

// NetworkPolicyRepository NetworkPolicy 数据访问接口
type NetworkPolicyRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.NetworkPolicy, error)
}

// ServiceAccountRepository ServiceAccount 数据访问接口
type ServiceAccountRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.ServiceAccount, error)
}

// GenericRepository 通用操作接口 (写操作 + 动态查询)
type GenericRepository interface {
	DeletePod(ctx context.Context, namespace, name string, opts model.DeleteOptions) error
	Delete(ctx context.Context, kind, namespace, name string, opts model.DeleteOptions) error
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error
	RestartDeployment(ctx context.Context, namespace, name string) error
	UpdateDeploymentImage(ctx context.Context, namespace, name, container, image string) error
	CordonNode(ctx context.Context, name string) error
	UncordonNode(ctx context.Context, name string) error
	GetConfigMapData(ctx context.Context, namespace, name string) (map[string]string, error)
	GetSecretData(ctx context.Context, namespace, name string) (map[string]string, error)
	Execute(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error)
}

// =============================================================================
// Metrics 仓库 — 实现: repository/metrics/
// =============================================================================

// MetricsRepository 节点硬件指标仓库接口
//
// 从 SDK ReceiverClient 拉取节点指标数据。
// 数据由 ReceiverClient 接收并暂存，本仓库负责按需拉取。
type MetricsRepository interface {
	GetAll() map[string]*model_v2.NodeMetricsSnapshot
}

// =============================================================================
// SLO 仓库 — 实现: repository/slo/
// =============================================================================

// SLORepository SLO 指标仓库接口
//
// 从 Ingress Controller 采集 SLO 指标，计算增量后返回。
type SLORepository interface {
	Collect(ctx context.Context) (*model_v2.SLOSnapshot, error)
	CollectRoutes(ctx context.Context) ([]model_v2.IngressRouteInfo, error)
}
