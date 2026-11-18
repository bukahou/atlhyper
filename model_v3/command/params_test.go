package command

import (
	"encoding/json"
	"reflect"
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
