package processor

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/datahub/memory"
	"AtlHyper/model_v3/cluster"
)

// 上报方即使没有清理，清单副本注解也不得进入存储。
func TestProcessSnapshot_DropsManifestCopyAnnotations(t *testing.T) {
	const lastApplied = "kubectl.kubernetes.io/last-applied-configuration"
	store := memory.NewMemoryStore(time.Hour, time.Minute, time.Hour)
	proc := NewProcessor(Config{Store: store})

	snapshot := &cluster.ClusterSnapshot{
		ClusterID: "c1",
		Deployments: []cluster.Deployment{{
			Annotations: map[string]string{lastApplied: `{"kind":"Deployment"}`, "team": "platform"},
			Template:    cluster.PodTemplate{Annotations: map[string]string{lastApplied: "{}"}},
		}},
	}
	if err := proc.ProcessSnapshot("c1", snapshot); err != nil {
		t.Fatalf("ProcessSnapshot: %v", err)
	}

	stored, err := store.GetSnapshot("c1")
	if err != nil || stored == nil {
		t.Fatalf("GetSnapshot: %v, %v", stored, err)
	}
	deployment := stored.Deployments[0]
	if _, ok := deployment.Annotations[lastApplied]; ok {
		t.Error("Deployment 的清单副本注解进入了存储")
	}
	if _, ok := deployment.Template.Annotations[lastApplied]; ok {
		t.Error("Pod 模板的清单副本注解进入了存储")
	}
	if deployment.Annotations["team"] != "platform" {
		t.Errorf("普通注解丢失: %v", deployment.Annotations)
	}
}
