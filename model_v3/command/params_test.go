package command

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// 发指令的一方用 ToParams 生成参数，执行的一方解析回同一个结构：两端不会各写各的键名。
func TestGetLogsParams_RoundTrip(t *testing.T) {
	sent := GetLogsParams{Container: "app", TailLines: 150, Previous: true}

	raw, err := json.Marshal(sent.ToParams()) // 指令经 JSON 传给 Agent
	if err != nil {
		t.Fatal(err)
	}
	var transported map[string]any
	if err := json.Unmarshal(raw, &transported); err != nil {
		t.Fatal(err)
	}

	var received GetLogsParams
	if err := ParseParams(transported, &received); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(received, sent) {
		t.Errorf("解析回来 %+v, want %+v", received, sent)
	}
}

func TestGetLogsParams_OmitsZeroValues(t *testing.T) {
	got := GetLogsParams{TailLines: 100}.ToParams()
	want := map[string]any{"tailLines": float64(100)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ToParams() = %#v, want %#v", got, want)
	}
}

func int32Ptr(v int32) *int32 { return &v }

func TestScaleParams_Validate(t *testing.T) {
	cases := []struct {
		name    string
		params  ScaleParams
		wantErr bool
	}{
		{"缺副本数（不能当成 0 去执行）", ScaleParams{}, true},
		{"负数", ScaleParams{Replicas: int32Ptr(-1)}, true},
		{"超过上限", ScaleParams{Replicas: int32Ptr(MaxScaleReplicas + 1)}, true},
		{"明确缩到 0", ScaleParams{Replicas: int32Ptr(0)}, false},
		{"正常", ScaleParams{Replicas: int32Ptr(3)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.params.Validate(); (err != nil) != c.wantErr {
				t.Fatalf("Validate() = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

// 参数走一遍 Master → map → Agent：缺字段在 Agent 那头仍然是「缺」，不会变成 0
func TestScaleParams_MissingReplicasSurvivesTransport(t *testing.T) {
	var parsed ScaleParams
	if err := ParseParams(map[string]any{}, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Validate() == nil {
		t.Fatal("没带副本数的参数应校验失败")
	}
	if err := ParseParams(ScaleParams{Replicas: int32Ptr(2)}.ToParams(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Replicas == nil || *parsed.Replicas != 2 {
		t.Fatalf("Replicas = %v, want 2", parsed.Replicas)
	}
}

func TestUpdateImageParams_Validate(t *testing.T) {
	cases := []struct {
		name    string
		params  UpdateImageParams
		wantErr bool
	}{
		{"短名字", UpdateImageParams{Container: "app", Image: "nginx"}, false},
		{"带标签", UpdateImageParams{Container: "app", Image: "nginx:1.27"}, false},
		{"完整仓库地址", UpdateImageParams{Container: "app", Image: "registry.example.com:5000/team/app:v1.0.0-abc1234"}, false},
		{"按摘要", UpdateImageParams{Container: "app", Image: "nginx@sha256:" + strings.Repeat("a", 64)}, false},
		{"缺容器名", UpdateImageParams{Image: "nginx"}, true},
		{"缺镜像", UpdateImageParams{Container: "app"}, true},
		{"注入 JSON", UpdateImageParams{Container: "app", Image: `nginx","securityContext":{"privileged":true},"x":"`}, true},
		{"带空格", UpdateImageParams{Container: "app", Image: "nginx latest"}, true},
		{"大写仓库名", UpdateImageParams{Container: "app", Image: "Nginx"}, true},
		{"容器名不合法", UpdateImageParams{Container: `app","image":"evil`, Image: "nginx"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.params.Validate(); (err != nil) != c.wantErr {
				t.Fatalf("Validate() = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}
