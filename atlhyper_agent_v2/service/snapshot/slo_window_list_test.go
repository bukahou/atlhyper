package snapshot

import (
	"testing"

	"AtlHyper/model_v3/slo"
)

// Agent 实际采集的窗口必须与共享的窗口列表一致：Master 与 AI 工具按那份列表校验参数
func TestSLOWindowConfigs_MatchSharedList(t *testing.T) {
	if len(sloWindowConfigs) != len(slo.Windows) {
		t.Fatalf("采集 %d 个窗口，共享列表 %d 个", len(sloWindowConfigs), len(slo.Windows))
	}
	for i, wc := range sloWindowConfigs {
		if wc.key != slo.Windows[i] {
			t.Errorf("第 %d 个窗口 = %q, want %q", i, wc.key, slo.Windows[i])
		}
	}
}
