package repository

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v2"
)

// persistentVolumeRepository PV 仓库实现
type persistentVolumeRepository struct {
	client sdk.K8sClient
}

// NewPersistentVolumeRepository 创建 PV 仓库
func NewPersistentVolumeRepository(client sdk.K8sClient) PersistentVolumeRepository {
	return &persistentVolumeRepository{client: client}
}

// List 列出 PersistentVolume
func (r *persistentVolumeRepository) List(ctx context.Context, opts model.ListOptions) ([]model_v2.PersistentVolume, error) {
	k8sPVs, err := r.client.ListPersistentVolumes(ctx, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	pvs := make([]model_v2.PersistentVolume, 0, len(k8sPVs))
	for i := range k8sPVs {
		pvs = append(pvs, ConvertPersistentVolume(&k8sPVs[i]))
	}
	return pvs, nil
}
