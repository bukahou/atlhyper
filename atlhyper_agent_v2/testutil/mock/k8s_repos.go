package mock

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/model_v3/cluster"
)

// =============================================================================
// NodeRepository mock
// =============================================================================

type NodeRepository struct {
	ListFn func(ctx context.Context, opts model.ListOptions) ([]cluster.Node, error)
	GetFn  func(ctx context.Context, name string) (*cluster.Node, error)
}

func (m *NodeRepository) List(ctx context.Context, opts model.ListOptions) ([]cluster.Node, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, opts)
	}
	return nil, nil
}

func (m *NodeRepository) Get(ctx context.Context, name string) (*cluster.Node, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, name)
	}
	return nil, nil
}

// =============================================================================
// DeploymentRepository mock
// =============================================================================

type DeploymentRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Deployment, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.Deployment, error)
}

func (m *DeploymentRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Deployment, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *DeploymentRepository) Get(ctx context.Context, namespace, name string) (*cluster.Deployment, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// StatefulSetRepository mock
// =============================================================================

type StatefulSetRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.StatefulSet, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.StatefulSet, error)
}

func (m *StatefulSetRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.StatefulSet, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *StatefulSetRepository) Get(ctx context.Context, namespace, name string) (*cluster.StatefulSet, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// DaemonSetRepository mock
// =============================================================================

type DaemonSetRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.DaemonSet, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.DaemonSet, error)
}

func (m *DaemonSetRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.DaemonSet, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *DaemonSetRepository) Get(ctx context.Context, namespace, name string) (*cluster.DaemonSet, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// ReplicaSetRepository mock
// =============================================================================

type ReplicaSetRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ReplicaSet, error)
}

func (m *ReplicaSetRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ReplicaSet, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

// =============================================================================
// ServiceRepository mock
// =============================================================================

type ServiceRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Service, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.Service, error)
}

func (m *ServiceRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Service, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *ServiceRepository) Get(ctx context.Context, namespace, name string) (*cluster.Service, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// IngressRepository mock
// =============================================================================

type IngressRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Ingress, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.Ingress, error)
}

func (m *IngressRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Ingress, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *IngressRepository) Get(ctx context.Context, namespace, name string) (*cluster.Ingress, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// ConfigMapRepository mock
// =============================================================================

type ConfigMapRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ConfigMap, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.ConfigMap, error)
}

func (m *ConfigMapRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ConfigMap, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *ConfigMapRepository) Get(ctx context.Context, namespace, name string) (*cluster.ConfigMap, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// SecretRepository mock
// =============================================================================

type SecretRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Secret, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.Secret, error)
}

func (m *SecretRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Secret, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *SecretRepository) Get(ctx context.Context, namespace, name string) (*cluster.Secret, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// NamespaceRepository mock
// =============================================================================

type NamespaceRepository struct {
	ListFn func(ctx context.Context, opts model.ListOptions) ([]cluster.Namespace, error)
	GetFn  func(ctx context.Context, name string) (*cluster.Namespace, error)
}

func (m *NamespaceRepository) List(ctx context.Context, opts model.ListOptions) ([]cluster.Namespace, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, opts)
	}
	return nil, nil
}

func (m *NamespaceRepository) Get(ctx context.Context, name string) (*cluster.Namespace, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, name)
	}
	return nil, nil
}

// =============================================================================
// EventRepository mock
// =============================================================================

type EventRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Event, error)
}

func (m *EventRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Event, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

// =============================================================================
// JobRepository mock
// =============================================================================

type JobRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Job, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.Job, error)
}

func (m *JobRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Job, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *JobRepository) Get(ctx context.Context, namespace, name string) (*cluster.Job, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// CronJobRepository mock
// =============================================================================

type CronJobRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.CronJob, error)
	GetFn  func(ctx context.Context, namespace, name string) (*cluster.CronJob, error)
}

func (m *CronJobRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.CronJob, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

func (m *CronJobRepository) Get(ctx context.Context, namespace, name string) (*cluster.CronJob, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}

// =============================================================================
// PersistentVolumeRepository mock
// =============================================================================

type PersistentVolumeRepository struct {
	ListFn func(ctx context.Context, opts model.ListOptions) ([]cluster.PersistentVolume, error)
}

func (m *PersistentVolumeRepository) List(ctx context.Context, opts model.ListOptions) ([]cluster.PersistentVolume, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, opts)
	}
	return nil, nil
}

// =============================================================================
// PersistentVolumeClaimRepository mock
// =============================================================================

type PersistentVolumeClaimRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.PersistentVolumeClaim, error)
}

func (m *PersistentVolumeClaimRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.PersistentVolumeClaim, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

// =============================================================================
// ResourceQuotaRepository mock
// =============================================================================

type ResourceQuotaRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ResourceQuota, error)
}

func (m *ResourceQuotaRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ResourceQuota, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

// =============================================================================
// LimitRangeRepository mock
// =============================================================================

type LimitRangeRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.LimitRange, error)
}

func (m *LimitRangeRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.LimitRange, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

// =============================================================================
// NetworkPolicyRepository mock
// =============================================================================

type NetworkPolicyRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.NetworkPolicy, error)
}

func (m *NetworkPolicyRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.NetworkPolicy, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}

// =============================================================================
// ServiceAccountRepository mock
// =============================================================================

type ServiceAccountRepository struct {
	ListFn func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ServiceAccount, error)
}

func (m *ServiceAccountRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ServiceAccount, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}
