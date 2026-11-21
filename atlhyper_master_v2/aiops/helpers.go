// atlhyper_master_v2/aiops/helpers.go
// AIOps 工具函数
package aiops

import (
	"strings"

	"AtlHyper/model_v3/slo"
)

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
		namespace = ClusterScope
	}
	return namespace + "/" + entityType + "/" + name
}

// ClusterScope 集群级对象（节点等）在实体键里占命名空间位置的占位
const ClusterScope = "_cluster"

// IngressEntityKey 入口（SLO）实体的键：按入口指标的服务标识（model_v3/slo.ServiceKey，"namespace/service"）
// 落在该服务的命名空间里，名称取服务名，与依赖图里 routes_to 的那一端对得上。
// ⛔ 不把整个服务标识塞进名称位置：标识里本身带斜杠，拆键时会拆错。
func IngressEntityKey(serviceKey string) string {
	ns, svc := slo.SplitServiceKey(serviceKey)
	return EntityKey(ns, "ingress", svc)
}

// SplitEntityKey 把实体键或主体键拆成命名空间、类型、名称。
// 集群级对象的命名空间返回空串；格式认不出时类型为空、名称为整个键。
func SplitEntityKey(key string) (namespace, entityType, name string) {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) != 3 {
		return "", "", key
	}
	namespace, entityType, name = parts[0], parts[1], parts[2]
	if namespace == ClusterScope {
		namespace = ""
	}
	return namespace, entityType, name
}
