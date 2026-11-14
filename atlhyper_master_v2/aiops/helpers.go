// atlhyper_master_v2/aiops/helpers.go
// AIOps 工具函数
package aiops

// EntityRef 指向某个集群里的某个实体。
//
// 实体键（namespace/type/name）只在一个集群内唯一；跨集群保存的状态（基线、状态机条目）
// 必须用 EntityRef 做键，否则两个集群的同名实体会共用同一份状态。
type EntityRef struct {
	ClusterID string
	EntityKey string
}

// EntityKey 生成实体唯一标识
// 格式: "namespace/type/name"
// 示例:
//
//	"default/pod/api-server-abc-123"
//	"default/service/api-server"
//	"_cluster/node/worker-3"
func EntityKey(namespace, entityType, name string) string {
	if namespace == "" {
		namespace = "_cluster"
	}
	return namespace + "/" + entityType + "/" + name
}
