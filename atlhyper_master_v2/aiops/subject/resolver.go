// atlhyper_master_v2/aiops/subject/resolver.go
// 主体解析与聚合：状态机与事件以「主体」为单位，有上级工作负载的 Pod 归到工作负载
package subject

import (
	"sort"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/cluster"
)

// 主体类型（主体键 "<ns>/<类型>/<名字>" 的中间一段）；其余实体的主体是自身，类型即实体类型
const (
	TypeDeployment  = "deployment"
	TypeStatefulSet = "statefulset"
	TypeDaemonSet   = "daemonset"
	TypeCronJob     = "cronjob"
	TypeJob         = "job"
)

// namespacedName 命名空间内的对象名，用作上级查找的键
type namespacedName struct {
	namespace string
	name      string
}

// ResolveSubjects 从快照得出「实体键 → 主体键」；不在返回值里的实体，主体就是它自己。
//
// 只有 Pod 会归到别的主体：
//   - 上级是 ReplicaSet 且该 ReplicaSet 属于某个 Deployment → Deployment
//   - 上级是 StatefulSet / DaemonSet → 该工作负载
//   - 上级是 Job → 所属 CronJob；Job 没有 CronJob 上级时归到 Job
//
// 没有上级、上级在快照里找不到的 Pod 不归并。
func ResolveSubjects(snap *cluster.ClusterSnapshot) map[string]string {
	subjects := make(map[string]string)
	if snap == nil {
		return subjects
	}

	deploymentOfReplicaSet := make(map[namespacedName]string, len(snap.ReplicaSets))
	for i := range snap.ReplicaSets {
		rs := &snap.ReplicaSets[i]
		if rs.OwnerKind == "Deployment" && rs.OwnerName != "" {
			deploymentOfReplicaSet[namespacedName{rs.Namespace, rs.Name}] = rs.OwnerName
		}
	}
	// 值为所属 CronJob 的名字；空串表示 Job 在快照里、但不属于 CronJob
	cronJobOfJob := make(map[namespacedName]string, len(snap.Jobs))
	for i := range snap.Jobs {
		job := &snap.Jobs[i]
		owner := ""
		if job.OwnerKind == "CronJob" {
			owner = job.OwnerName
		}
		cronJobOfJob[namespacedName{job.Namespace, job.Name}] = owner
	}

	for i := range snap.Pods {
		pod := &snap.Pods[i].Summary
		if pod.OwnerName == "" {
			continue
		}
		namespace := pod.Namespace
		owner := namespacedName{namespace, pod.OwnerName}

		var subjectKey string
		switch pod.OwnerKind {
		case "ReplicaSet":
			if deployment, ok := deploymentOfReplicaSet[owner]; ok {
				subjectKey = aiops.EntityKey(namespace, TypeDeployment, deployment)
			}
		case "StatefulSet":
			subjectKey = aiops.EntityKey(namespace, TypeStatefulSet, pod.OwnerName)
		case "DaemonSet":
			subjectKey = aiops.EntityKey(namespace, TypeDaemonSet, pod.OwnerName)
		case "Job":
			if cronJob, ok := cronJobOfJob[owner]; ok {
				if cronJob != "" {
					subjectKey = aiops.EntityKey(namespace, TypeCronJob, cronJob)
				} else {
					subjectKey = aiops.EntityKey(namespace, TypeJob, pod.OwnerName)
				}
			}
		}
		if subjectKey != "" {
			subjects[aiops.EntityKey(namespace, "pod", pod.Name)] = subjectKey
		}
	}
	return subjects
}

// AggregateBySubject 把实体风险按主体聚合。
//
// 主体的风险取成员中最高的 RFinal，该成员为领头实体（风险相同时取键最小的，结果稳定）。
// 成员全部无风险的主体也在返回值里：状态机靠它判断恢复，并据此知道主体仍然存在。
func AggregateBySubject(entityRisks map[string]*aiops.EntityRisk, subjects map[string]string) map[string]*aiops.SubjectRisk {
	result := make(map[string]*aiops.SubjectRisk)

	for entityKey, risk := range entityRisks {
		if risk == nil {
			continue
		}
		subjectKey, ok := subjects[entityKey]
		if !ok {
			subjectKey = entityKey
		}

		sr := result[subjectKey]
		if sr == nil {
			sr = &aiops.SubjectRisk{SubjectKey: subjectKey}
			result[subjectKey] = sr
		}
		if sr.Lead == nil || ranksBefore(risk, sr.Lead) {
			sr.Lead = risk
			sr.RFinal = risk.RFinal
		}
		if risk.RFinal > 0 {
			sr.Members = append(sr.Members, risk)
		}
	}

	for _, sr := range result {
		sort.Slice(sr.Members, func(i, j int) bool { return ranksBefore(sr.Members[i], sr.Members[j]) })
	}
	return result
}

// ranksBefore 风险高的在前；风险相同按实体键升序
func ranksBefore(a, b *aiops.EntityRisk) bool {
	if a.RFinal != b.RFinal {
		return a.RFinal > b.RFinal
	}
	return a.EntityKey < b.EntityKey
}
