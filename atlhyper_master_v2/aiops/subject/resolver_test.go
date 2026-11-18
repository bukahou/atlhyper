package subject

import (
	"reflect"
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

func ownedPod(namespace, name, ownerKind, ownerName string) cluster.Pod {
	return cluster.Pod{Summary: cluster.PodSummary{
		Namespace: namespace, Name: name, OwnerKind: ownerKind, OwnerName: ownerName,
	}}
}

func ownedMeta(namespace, name, ownerKind, ownerName string) model_v3.CommonMeta {
	return model_v3.CommonMeta{Namespace: namespace, Name: name, OwnerKind: ownerKind, OwnerName: ownerName}
}

func TestResolveSubjects(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			ownedPod("shop", "api-7d9f-abc", "ReplicaSet", "api-7d9f"),
			ownedPod("shop", "api-7d9f-def", "ReplicaSet", "api-7d9f"),
			ownedPod("shop", "db-0", "StatefulSet", "db"),
			ownedPod("kube-system", "proxy-x1", "DaemonSet", "proxy"),
			ownedPod("shop", "report-2900-k1", "Job", "report-2900"),
			ownedPod("shop", "migrate-z9", "Job", "migrate"),
			ownedPod("shop", "debug", "", ""),
			ownedPod("shop", "orphan-rs-p1", "ReplicaSet", "orphan-rs"),
			ownedPod("shop", "gone-rs-p1", "ReplicaSet", "gone-rs"),
			ownedPod("shop", "gone-job-p1", "Job", "gone-job"),
			ownedPod("kube-system", "etcd-node-1", "Node", "node-1"),
			// 另一个命名空间里的同名 ReplicaSet 属于另一个 Deployment
			ownedPod("staging", "api-7d9f-abc", "ReplicaSet", "api-7d9f"),
		},
		ReplicaSets: []cluster.ReplicaSet{
			{CommonMeta: ownedMeta("shop", "api-7d9f", "Deployment", "api")},
			{CommonMeta: ownedMeta("shop", "orphan-rs", "", "")},
			{CommonMeta: ownedMeta("staging", "api-7d9f", "Deployment", "api-canary")},
		},
		Jobs: []cluster.Job{
			{CommonMeta: ownedMeta("shop", "report-2900", "CronJob", "report")},
			{CommonMeta: ownedMeta("shop", "migrate", "", "")},
		},
	}

	got := ResolveSubjects(snap)

	want := map[string]string{
		"shop/pod/api-7d9f-abc":    "shop/deployment/api",
		"shop/pod/api-7d9f-def":    "shop/deployment/api",
		"shop/pod/db-0":            "shop/statefulset/db",
		"kube-system/pod/proxy-x1": "kube-system/daemonset/proxy",
		"shop/pod/report-2900-k1":  "shop/cronjob/report",
		"shop/pod/migrate-z9":      "shop/job/migrate",
		"staging/pod/api-7d9f-abc": "staging/deployment/api-canary",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ResolveSubjects()\n got  %v\n want %v", got, want)
	}

	// 没有上级、上级不属于任何 Deployment、上级不在快照里、上级不是工作负载：主体是自身（不出现在返回值里）
	for _, self := range []string{
		"shop/pod/debug", "shop/pod/orphan-rs-p1", "shop/pod/gone-rs-p1", "shop/pod/gone-job-p1", "kube-system/pod/etcd-node-1",
	} {
		if subjectKey, ok := got[self]; ok {
			t.Errorf("%s 的主体应是自身，实际归到 %s", self, subjectKey)
		}
	}
}

func TestResolveSubjects_NilSnapshot(t *testing.T) {
	if got := ResolveSubjects(nil); len(got) != 0 {
		t.Errorf("空快照应返回空映射，实际 %v", got)
	}
}

func entityRisk(key string, rFinal float64) *aiops.EntityRisk {
	return &aiops.EntityRisk{EntityKey: key, EntityType: aiops.ExtractEntityType(key), RFinal: rFinal}
}

func memberKeys(members []*aiops.EntityRisk) []string {
	keys := make([]string, len(members))
	for i, m := range members {
		keys[i] = m.EntityKey
	}
	return keys
}

func TestAggregateBySubject(t *testing.T) {
	subjects := map[string]string{
		"shop/pod/api-a": "shop/deployment/api",
		"shop/pod/api-b": "shop/deployment/api",
		"shop/pod/api-c": "shop/deployment/api",
		"shop/pod/web-a": "shop/deployment/web",
		"shop/pod/web-b": "shop/deployment/web",
	}
	entityRisks := map[string]*aiops.EntityRisk{
		"shop/pod/api-a":     entityRisk("shop/pod/api-a", 0.3),
		"shop/pod/api-b":     entityRisk("shop/pod/api-b", 0.7),
		"shop/pod/api-c":     entityRisk("shop/pod/api-c", 0),
		"shop/pod/web-a":     entityRisk("shop/pod/web-a", 0),
		"shop/pod/web-b":     entityRisk("shop/pod/web-b", 0),
		"shop/service/api":   entityRisk("shop/service/api", 0.4),
		"_cluster/node/n1":   entityRisk("_cluster/node/n1", 0),
		"shop/pod/nil-entry": nil,
	}

	got := AggregateBySubject(entityRisks, subjects)

	if len(got) != 4 {
		t.Fatalf("期望 4 个主体，实际 %d 个: %v", len(got), got)
	}

	api := got["shop/deployment/api"]
	if api == nil {
		t.Fatal("缺少主体 shop/deployment/api")
	}
	if api.SubjectKey != "shop/deployment/api" || api.RFinal != 0.7 || api.Lead.EntityKey != "shop/pod/api-b" {
		t.Errorf("工作负载主体：风险取成员最高值、领头是风险最高的成员，实际 %+v lead=%+v", api, api.Lead)
	}
	if keys := memberKeys(api.Members); !reflect.DeepEqual(keys, []string{"shop/pod/api-b", "shop/pod/api-a"}) {
		t.Errorf("成员只含有风险的，按风险从高到低，实际 %v", keys)
	}

	// 成员全部无风险的主体仍要参与评估（状态机靠它判断恢复、刷新最近评估时刻）
	web := got["shop/deployment/web"]
	if web == nil || web.RFinal != 0 || web.Lead == nil || len(web.Members) != 0 {
		t.Fatalf("无风险的主体应存在、风险为 0、有领头、成员为空，实际 %+v", web)
	}
	if web.Lead.EntityKey != "shop/pod/web-a" {
		t.Errorf("风险相同时领头取键最小的（结果稳定），实际 %s", web.Lead.EntityKey)
	}

	// 不在映射里的实体，主体是它自己
	svc := got["shop/service/api"]
	if svc == nil || svc.RFinal != 0.4 || svc.Lead.EntityKey != "shop/service/api" || len(svc.Members) != 1 {
		t.Errorf("service 的主体应是自身，实际 %+v", svc)
	}
	if node := got["_cluster/node/n1"]; node == nil || node.RFinal != 0 {
		t.Errorf("node 的主体应是自身，实际 %+v", node)
	}
}
