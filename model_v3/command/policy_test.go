package command

import (
	"errors"
	"testing"
)

func TestCheckSource(t *testing.T) {
	cases := []struct {
		name   string
		action string
		source string
		want   bool
	}{
		// 部署器只能应用清单；应用清单也只有部署器能发
		{"部署器应用清单", ActionApplyManifests, SourceDeployer, true},
		{"Web 应用清单", ActionApplyManifests, SourceWeb, false},
		{"AI 应用清单", ActionApplyManifests, SourceAI, false},
		{"部署器扩缩容", ActionScale, SourceDeployer, false},

		// AI 只许读
		{"AI 读日志", ActionGetLogs, SourceAI, true},
		{"AI 动态只读查询", ActionDynamic, SourceAI, true},
		{"AI 查链路", ActionQueryTraces, SourceAI, true},
		{"AI 扩缩容", ActionScale, SourceAI, false},
		{"AI 重启", ActionRestart, SourceAI, false},
		{"AI 删除", ActionDelete, SourceAI, false},
		{"AI 改镜像", ActionUpdateImage, SourceAI, false},
		{"AI cordon", ActionCordon, SourceAI, false},
		{"AI 读 Secret", ActionGetSecret, SourceAI, false},

		// Web 的操作都走各自的专用接口
		{"Web 扩缩容", ActionScale, SourceWeb, true},
		{"Web 读 Secret", ActionGetSecret, SourceWeb, true},
		{"Web 动态查询", ActionDynamic, SourceWeb, false},

		// 缺来源、未知来源、未知动作一律拒绝
		{"没有来源", ActionGetLogs, "", false},
		{"未知来源", ActionGetLogs, "cli", false},
		{"未知动作", "exec", SourceWeb, false},
		{"空动作", "", SourceWeb, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckSource(c.action, c.source)
			if (err == nil) != c.want {
				t.Fatalf("CheckSource(%q, %q) = %v, want permitted=%v", c.action, c.source, err, c.want)
			}
			if err != nil && !errors.Is(err, ErrActionNotPermitted) {
				t.Errorf("拒绝时应返回 ErrActionNotPermitted，实际 %v", err)
			}
		})
	}
}

// 每个动作常量都必须在名单里声明来源：漏了等于这个动作谁都发不了，写了没接线
func TestActionSources_CoversEveryAction(t *testing.T) {
	all := []string{
		ActionScale, ActionRestart, ActionDelete, ActionCordon, ActionUncordon,
		ActionUpdateImage, ActionGetLogs, ActionGetConfigMap, ActionGetSecret,
		ActionDynamic, ActionApplyManifests,
		ActionQueryTraces, ActionQueryTraceDetail, ActionQueryLogs, ActionQueryMetrics,
	}
	for _, action := range all {
		if len(actionSources[action]) == 0 {
			t.Errorf("动作 %q 没有声明允许的来源", action)
		}
	}
	if len(actionSources) != len(all) {
		t.Errorf("名单里有 %d 个动作，测试列了 %d 个：新增动作时两边一起改", len(actionSources), len(all))
	}
}

func TestIsValidAction(t *testing.T) {
	if !IsValidAction(ActionRestart) {
		t.Error("restart 应是有效动作")
	}
	for _, removed := range []string{"exec", "drain", "delete_pod", "query_slo", ""} {
		if IsValidAction(removed) {
			t.Errorf("%q 不应是有效动作", removed)
		}
	}
}
