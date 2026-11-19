package k8s

import (
	"encoding/json"
	"testing"
	"time"
)

// 镜像字段里即使带引号和花括号，生成的 patch 也只改那一个容器的 image，不会多出别的字段
func TestImagePatch_ContentStaysAValue(t *testing.T) {
	raw, err := imagePatch("app", `nginx","securityContext":{"privileged":true},"x":"`)
	if err != nil {
		t.Fatal(err)
	}
	var patch struct {
		Spec struct {
			Template struct {
				Spec struct {
					Containers []map[string]any `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(raw, &patch); err != nil {
		t.Fatalf("patch 不是合法 JSON: %v\n%s", err, raw)
	}
	containers := patch.Spec.Template.Spec.Containers
	if len(containers) != 1 || len(containers[0]) != 2 {
		t.Fatalf("patch 应只有一个容器、两个字段，实际 %s", raw)
	}
	if _, injected := containers[0]["securityContext"]; injected {
		t.Fatalf("镜像内容被解释成了字段: %s", raw)
	}
}

func TestRestartPatch(t *testing.T) {
	raw, err := restartPatch(time.Date(2026, 11, 19, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"2026-11-19T01:02:03Z"}}}}}`
	if string(raw) != want {
		t.Errorf("patch = %s\nwant    %s", raw, want)
	}
}
