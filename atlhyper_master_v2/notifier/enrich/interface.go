// atlhyper_master_v2/notifier/enrich/interface.go
// 资源查询接口定义
package enrich

import (
	"context"

	"AtlHyper/model_v2"
)

// ResourceQuery 资源查询接口
// 由 service/query 实现，notifier 通过此接口获取资源详情
// 不直接访问 DataHub
type ResourceQuery interface {
	GetPod(ctx context.Context, clusterID, namespace, name string) (*model_v2.Pod, error)
	GetNode(ctx context.Context, clusterID, name string) (*model_v2.Node, error)
	GetDeployment(ctx context.Context, clusterID, namespace, name string) (*model_v2.Deployment, error)
	GetDeploymentByReplicaSet(ctx context.Context, clusterID, namespace, rsName string) (*model_v2.Deployment, error)
}
