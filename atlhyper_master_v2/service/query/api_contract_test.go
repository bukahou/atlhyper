package query

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"AtlHyper/api/contract"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
)

// 代理什么都没采到时（数组缺省、读数为零），接口输出仍须符合规格：数组是数组（⛔ null），等级在枚举里
var sparseNodes = []metrics.NodeMetrics{
	{NodeName: "bare"},
	{NodeName: "with-hardware", Hardware: &metrics.NodeHardware{}},
}

func TestNodeMetricsViews_SparseDataMatchesSpec(t *testing.T) {
	body, err := json.Marshal(map[string]any{"message": "ok", "data": nodeViews(t, sparseNodes...)})
	if err != nil {
		t.Fatal(err)
	}
	contract.AssertValid(t, http.MethodGet, "/api/v2/observe/metrics/nodes", http.StatusOK, body)
}

func TestHardwareHealth_SparseDataMatchesSpec(t *testing.T) {
	store := &mockStoreForOverview{snapshots: map[string]*cluster.ClusterSnapshot{
		"c1": {OTel: &cluster.OTelSnapshot{MetricsNodes: sparseNodes}},
	}}
	health, err := (&OTelQuery{store: store}).GetHardwareHealth(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{"message": "ok", "data": health})
	if err != nil {
		t.Fatal(err)
	}
	contract.AssertValid(t, http.MethodGet, "/api/v2/observe/metrics/hardware", http.StatusOK, body)
}
