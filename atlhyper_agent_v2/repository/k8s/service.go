package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/model_v2"
)

// serviceRepository Service 仓库实现
type serviceRepository struct {
	client sdk.K8sClient
}

// NewServiceRepository 创建 Service 仓库
func NewServiceRepository(client sdk.K8sClient) repository.ServiceRepository {
	return &serviceRepository{client: client}
}

// List 列出 Service
func (r *serviceRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]model_v2.Service, error) {
	k8sServices, err := r.client.ListServices(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	services := make([]model_v2.Service, 0, len(k8sServices))
	for i := range k8sServices {
		services = append(services, ConvertService(&k8sServices[i]))
	}
	return services, nil
}

// Get 获取单个 Service
func (r *serviceRepository) Get(ctx context.Context, namespace, name string) (*model_v2.Service, error) {
	k8sService, err := r.client.GetService(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	service := ConvertService(k8sService)
	return &service, nil
}
