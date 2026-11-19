package query

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
)

// 节点没有这个读数：判成「未知」，⛔ 不能判成「正常」
func TestTempCompareCell_MissingIsUnknown(t *testing.T) {
	if got := tempCompareCell(nil, "°C").Status; got != model.HardwareUnknown {
		t.Errorf("缺读数时 status = %q, want unknown", got)
	}
}
