package mock

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/model_v3/cluster"
)

// PodRepository mock
type PodRepository struct {
	ListFn    func(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Pod, error)
	GetFn     func(ctx context.Context, namespace, name string) (*cluster.Pod, error)
	GetLogsFn func(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error)
}

func (m *PodRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Pod, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, namespace, opts)
	}
	return nil, nil
}
func (m *PodRepository) Get(ctx context.Context, namespace, name string) (*cluster.Pod, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, namespace, name)
	}
	return nil, nil
}
func (m *PodRepository) GetLogs(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error) {
	if m.GetLogsFn != nil {
		return m.GetLogsFn(ctx, namespace, name, opts)
	}
	return "", nil
}

// GenericRepository mock
type GenericRepository struct {
	DeletePodFn             func(ctx context.Context, namespace, name string, opts model.DeleteOptions) error
	DeleteFn                func(ctx context.Context, kind, namespace, name string, opts model.DeleteOptions) error
	ScaleDeploymentFn       func(ctx context.Context, namespace, name string, replicas int32) error
	RestartDeploymentFn     func(ctx context.Context, namespace, name string) error
	UpdateDeploymentImageFn func(ctx context.Context, namespace, name, container, image string) error
	CordonNodeFn            func(ctx context.Context, name string) error
	UncordonNodeFn          func(ctx context.Context, name string) error
	GetConfigMapDataFn      func(ctx context.Context, namespace, name string) (map[string]string, error)
	GetSecretDataFn         func(ctx context.Context, namespace, name string) (map[string]string, error)
	ExecuteFn               func(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error)
}

func (m *GenericRepository) DeletePod(ctx context.Context, namespace, name string, opts model.DeleteOptions) error {
	if m.DeletePodFn != nil {
		return m.DeletePodFn(ctx, namespace, name, opts)
	}
	return nil
}

func (m *GenericRepository) Delete(ctx context.Context, kind, namespace, name string, opts model.DeleteOptions) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, kind, namespace, name, opts)
	}
	return nil
}

func (m *GenericRepository) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	if m.ScaleDeploymentFn != nil {
		return m.ScaleDeploymentFn(ctx, namespace, name, replicas)
	}
	return nil
}

func (m *GenericRepository) RestartDeployment(ctx context.Context, namespace, name string) error {
	if m.RestartDeploymentFn != nil {
		return m.RestartDeploymentFn(ctx, namespace, name)
	}
	return nil
}

func (m *GenericRepository) UpdateDeploymentImage(ctx context.Context, namespace, name, container, image string) error {
	if m.UpdateDeploymentImageFn != nil {
		return m.UpdateDeploymentImageFn(ctx, namespace, name, container, image)
	}
	return nil
}

func (m *GenericRepository) CordonNode(ctx context.Context, name string) error {
	if m.CordonNodeFn != nil {
		return m.CordonNodeFn(ctx, name)
	}
	return nil
}

func (m *GenericRepository) UncordonNode(ctx context.Context, name string) error {
	if m.UncordonNodeFn != nil {
		return m.UncordonNodeFn(ctx, name)
	}
	return nil
}

func (m *GenericRepository) GetConfigMapData(ctx context.Context, namespace, name string) (map[string]string, error) {
	if m.GetConfigMapDataFn != nil {
		return m.GetConfigMapDataFn(ctx, namespace, name)
	}
	return nil, nil
}

func (m *GenericRepository) GetSecretData(ctx context.Context, namespace, name string) (map[string]string, error) {
	if m.GetSecretDataFn != nil {
		return m.GetSecretDataFn(ctx, namespace, name)
	}
	return nil, nil
}

func (m *GenericRepository) Execute(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error) {
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, req)
	}
	return nil, nil
}
