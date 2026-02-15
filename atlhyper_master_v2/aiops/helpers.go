// atlhyper_master_v2/aiops/helpers.go
// AIOps 工具函数
package aiops

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
