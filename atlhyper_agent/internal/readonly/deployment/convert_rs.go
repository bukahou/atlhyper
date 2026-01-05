// internal/readonly/deployment/convert_rs.go
package deployment

import (
	modeldep "AtlHyper/model/k8s"
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// buildReplicaSetIndex —— 全集群拉取 ReplicaSets，按 ownerUID 建索引
// 返回：map[deploymentUID][]ReplicaSet、总 RS 数、错误
func buildReplicaSetIndex(ctx context.Context, cs kubernetes.Interface) (map[types.UID][]appsv1.ReplicaSet, int, error) {
	rsList, err := cs.AppsV1().ReplicaSets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil || rsList == nil {
		return nil, 0, err
	}
	idx := make(map[types.UID][]appsv1.ReplicaSet, 256)
	for i := range rsList.Items {
		rs := rsList.Items[i]
		owner := firstControllerOwner(rs.ObjectMeta.OwnerReferences)
		if owner == nil || owner.Kind != "Deployment" || owner.UID == "" {
			continue
		}
		uid := owner.UID
		idx[uid] = append(idx[uid], rs)
	}
	return idx, len(rsList.Items), nil
}

func firstControllerOwner(ors []metav1.OwnerReference) *metav1.OwnerReference {
	for i := range ors {
		or := ors[i]
		if or.Controller != nil && *or.Controller {
			return &or
		}
	}
	if len(ors) > 0 {
		return &ors[0]
	}
	return nil
}

func rsBriefs(depUID types.UID, idx map[types.UID][]appsv1.ReplicaSet) []modeldep.ReplicaSetBrief {
	if idx == nil {
		return nil
	}
	list := idx[depUID]
	if len(list) == 0 {
		return nil
	}
	out := make([]modeldep.ReplicaSetBrief, 0, len(list))
	for i := range list {
		rs := &list[i]
		out = append(out, modeldep.ReplicaSetBrief{
			Name:      rs.Name,
			Namespace: rs.Namespace,
			Revision:  rs.Annotations["deployment.kubernetes.io/revision"],
			Replicas:  rs.Status.Replicas,
			Ready:     rs.Status.ReadyReplicas,
			Available: rs.Status.AvailableReplicas,
			CreatedAt: rs.CreationTimestamp.Time,
			Age:       fmtAge(rs.CreationTimestamp.Time),
		})
	}
	return out
}
