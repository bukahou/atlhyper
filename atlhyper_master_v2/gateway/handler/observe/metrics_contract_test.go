package observe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"AtlHyper/api/contract"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/metrics"
)

// 只实现契约测试用到的两个查询，其余方法不会被调用
type metricsQueryStub struct {
	ObserveQuery
	views  []model.NodeMetricsView
	health *model.HardwareHealthResponse
}

func (s metricsQueryStub) ListNodeMetricsViews(context.Context, string) ([]model.NodeMetricsView, error) {
	return s.views, nil
}

func (s metricsQueryStub) GetHardwareHealth(context.Context, string) (*model.HardwareHealthResponse, error) {
	return s.health, nil
}

var metricsEnumSamples = []any{model.HardwareGood, metrics.ProfileRaspi5}

func serveMetrics(t *testing.T, h http.HandlerFunc, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, path+"?cluster_id=c1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	return rec
}

// 节点指标列表：每个字段都填上值，响应须与规格逐项一致
func TestMetricsNodes_MatchesSpec(t *testing.T) {
	var view model.NodeMetricsView
	contract.Fill(&view, metricsEnumSamples...)
	h := NewObserveHandler(nil, metricsQueryStub{views: []model.NodeMetricsView{view}})

	rec := serveMetrics(t, h.MetricsNodes, "/api/v2/observe/metrics/nodes")
	contract.AssertResponse(t, http.MethodGet, "/api/v2/observe/metrics/nodes", http.StatusOK, rec.Body.Bytes())
}

// 硬件健康矩阵：同上
func TestMetricsHardware_MatchesSpec(t *testing.T) {
	var health model.HardwareHealthResponse
	contract.Fill(&health, metricsEnumSamples...)
	for i := range health.Rows {
		for j := range health.Rows[i].Sensors {
			health.Rows[i].Sensors[j].Class = string(metrics.TempClassDisk)
		}
	}
	h := NewObserveHandler(nil, metricsQueryStub{health: &health})

	rec := serveMetrics(t, h.MetricsHardware, "/api/v2/observe/metrics/hardware")
	contract.AssertResponse(t, http.MethodGet, "/api/v2/observe/metrics/hardware", http.StatusOK, rec.Body.Bytes())
}
