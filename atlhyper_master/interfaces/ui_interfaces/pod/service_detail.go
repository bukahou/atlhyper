package pod

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master/interfaces/datasource"
)

// GetPodDetail —— 根据 clusterID + namespace + podName 返回单个 Pod 的完整详情。
// 数据来源：GetPodListLatest（全集群列表），本函数做过滤。
// 返回类型：datasource.Pod（即底层 model/pod.Pod 的别名）——包含 Summary/Spec/Containers/Volumes/Network/Metrics。
func GetPodDetail(ctx context.Context, clusterID, namespace, podName string) (*PodDetailDTO, error) {
    pods, err := datasource.GetPodListLatest(ctx, clusterID)
    if err != nil {
        return nil, fmt.Errorf("get pod list failed: %w", err)
    }
    for _, p := range pods {
        if p.Summary.Namespace == namespace && p.Summary.Name == podName {
            dto := FromModel(p)
            return &dto, nil
        }
    }
    return nil, fmt.Errorf("pod not found: %s/%s (cluster=%s)", namespace, podName, clusterID)
}
