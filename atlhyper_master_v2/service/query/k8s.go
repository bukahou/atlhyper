// atlhyper_master_v2/service/query/k8s.go
// K8s 资源快照查询实现
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// ==================== 快照查询 ====================

// GetSnapshot 获取集群快照
func (q *QueryService) GetSnapshot(ctx context.Context, clusterID string) (*model_v2.ClusterSnapshot, error) {
	return q.store.GetSnapshot(clusterID)
}

// GetPods 获取 Pod 列表
func (q *QueryService) GetPods(ctx context.Context, clusterID string, opts model.PodQueryOpts) ([]model_v2.Pod, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	// 过滤
	result := make([]model_v2.Pod, 0)
	for _, pod := range snapshot.Pods {
		if opts.Namespace != "" && pod.GetNamespace() != opts.Namespace {
			continue
		}
		if opts.NodeName != "" && pod.GetNodeName() != opts.NodeName {
			continue
		}
		if opts.Phase != "" && pod.Status.Phase != opts.Phase {
			continue
		}

		// 格式化 metrics 单位
		pod.Status.CPUUsage = FormatCPU(pod.Status.CPUUsage)
		pod.Status.MemoryUsage = FormatMemory(pod.Status.MemoryUsage)

		result = append(result, pod)
	}

	// 分页
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// GetNodes 获取 Node 列表
func (q *QueryService) GetNodes(ctx context.Context, clusterID string) ([]model_v2.Node, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.Nodes, nil
}

// GetDeployments 获取 Deployment 列表
func (q *QueryService) GetDeployments(ctx context.Context, clusterID string, namespace string) ([]model_v2.Deployment, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Deployments, nil
	}

	result := make([]model_v2.Deployment, 0)
	for _, d := range snapshot.Deployments {
		if d.GetNamespace() == namespace {
			result = append(result, d)
		}
	}
	return result, nil
}

// GetServices 获取 Service 列表
func (q *QueryService) GetServices(ctx context.Context, clusterID string, namespace string) ([]model_v2.Service, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Services, nil
	}

	result := make([]model_v2.Service, 0)
	for _, s := range snapshot.Services {
		if s.GetNamespace() == namespace {
			result = append(result, s)
		}
	}
	return result, nil
}

// GetIngresses 获取 Ingress 列表
func (q *QueryService) GetIngresses(ctx context.Context, clusterID string, namespace string) ([]model_v2.Ingress, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Ingresses, nil
	}

	result := make([]model_v2.Ingress, 0)
	for _, i := range snapshot.Ingresses {
		if i.GetNamespace() == namespace {
			result = append(result, i)
		}
	}
	return result, nil
}

// GetConfigMaps 获取 ConfigMap 列表
func (q *QueryService) GetConfigMaps(ctx context.Context, clusterID string, namespace string) ([]model_v2.ConfigMap, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.ConfigMaps, nil
	}

	result := make([]model_v2.ConfigMap, 0)
	for _, c := range snapshot.ConfigMaps {
		if c.Namespace == namespace {
			result = append(result, c)
		}
	}
	return result, nil
}

// GetSecrets 获取 Secret 列表
func (q *QueryService) GetSecrets(ctx context.Context, clusterID string, namespace string) ([]model_v2.Secret, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Secrets, nil
	}

	result := make([]model_v2.Secret, 0)
	for _, s := range snapshot.Secrets {
		if s.Namespace == namespace {
			result = append(result, s)
		}
	}
	return result, nil
}

// GetNamespaces 获取 Namespace 列表
func (q *QueryService) GetNamespaces(ctx context.Context, clusterID string) ([]model_v2.Namespace, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.Namespaces, nil
}

// GetDaemonSets 获取 DaemonSet 列表
func (q *QueryService) GetDaemonSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.DaemonSet, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.DaemonSets, nil
	}

	result := make([]model_v2.DaemonSet, 0)
	for _, d := range snapshot.DaemonSets {
		if d.GetNamespace() == namespace {
			result = append(result, d)
		}
	}
	return result, nil
}

// GetStatefulSets 获取 StatefulSet 列表
func (q *QueryService) GetStatefulSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.StatefulSet, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.StatefulSets, nil
	}

	result := make([]model_v2.StatefulSet, 0)
	for _, s := range snapshot.StatefulSets {
		if s.GetNamespace() == namespace {
			result = append(result, s)
		}
	}
	return result, nil
}

// GetJobs 获取 Job 列表
func (q *QueryService) GetJobs(ctx context.Context, clusterID string, namespace string) ([]model_v2.Job, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Jobs, nil
	}

	result := make([]model_v2.Job, 0)
	for _, j := range snapshot.Jobs {
		if j.Namespace == namespace {
			result = append(result, j)
		}
	}
	return result, nil
}

// GetCronJobs 获取 CronJob 列表
func (q *QueryService) GetCronJobs(ctx context.Context, clusterID string, namespace string) ([]model_v2.CronJob, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.CronJobs, nil
	}

	result := make([]model_v2.CronJob, 0)
	for _, c := range snapshot.CronJobs {
		if c.Namespace == namespace {
			result = append(result, c)
		}
	}
	return result, nil
}

// GetPersistentVolumes 获取 PV 列表（集群级，无 namespace）
func (q *QueryService) GetPersistentVolumes(ctx context.Context, clusterID string) ([]model_v2.PersistentVolume, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.PersistentVolumes, nil
}

// GetPersistentVolumeClaims 获取 PVC 列表
func (q *QueryService) GetPersistentVolumeClaims(ctx context.Context, clusterID string, namespace string) ([]model_v2.PersistentVolumeClaim, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.PersistentVolumeClaims, nil
	}

	result := make([]model_v2.PersistentVolumeClaim, 0)
	for _, p := range snapshot.PersistentVolumeClaims {
		if p.Namespace == namespace {
			result = append(result, p)
		}
	}
	return result, nil
}

// GetNetworkPolicies 获取 NetworkPolicy 列表
func (q *QueryService) GetNetworkPolicies(ctx context.Context, clusterID string, namespace string) ([]model_v2.NetworkPolicy, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.NetworkPolicies, nil
	}

	result := make([]model_v2.NetworkPolicy, 0)
	for _, np := range snapshot.NetworkPolicies {
		if np.Namespace == namespace {
			result = append(result, np)
		}
	}
	return result, nil
}

// GetResourceQuotas 获取 ResourceQuota 列表
func (q *QueryService) GetResourceQuotas(ctx context.Context, clusterID string, namespace string) ([]model_v2.ResourceQuota, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.ResourceQuotas, nil
	}

	result := make([]model_v2.ResourceQuota, 0)
	for _, rq := range snapshot.ResourceQuotas {
		if rq.Namespace == namespace {
			result = append(result, rq)
		}
	}
	return result, nil
}

// GetLimitRanges 获取 LimitRange 列表
func (q *QueryService) GetLimitRanges(ctx context.Context, clusterID string, namespace string) ([]model_v2.LimitRange, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.LimitRanges, nil
	}

	result := make([]model_v2.LimitRange, 0)
	for _, lr := range snapshot.LimitRanges {
		if lr.Namespace == namespace {
			result = append(result, lr)
		}
	}
	return result, nil
}

// GetServiceAccounts 获取 ServiceAccount 列表
func (q *QueryService) GetServiceAccounts(ctx context.Context, clusterID string, namespace string) ([]model_v2.ServiceAccount, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.ServiceAccounts, nil
	}

	result := make([]model_v2.ServiceAccount, 0)
	for _, sa := range snapshot.ServiceAccounts {
		if sa.Namespace == namespace {
			result = append(result, sa)
		}
	}
	return result, nil
}
