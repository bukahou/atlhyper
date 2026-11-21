package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"AtlHyper/api/contract"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/model_v3/cluster"
)

// 只实现契约测试用到的查询，其余方法不会被调用
type contractQueryStub struct {
	service.Query
	overview *cluster.ClusterOverview
	events   []cluster.Event
	history  []*database.ClusterEvent
}

func (s contractQueryStub) GetOverview(context.Context, string) (*cluster.ClusterOverview, error) {
	return s.overview, nil
}
func (s contractQueryStub) GetEvents(context.Context, string, model.EventQueryOpts) ([]cluster.Event, error) {
	return s.events, nil
}
func (s contractQueryStub) ListEventHistory(context.Context, string, database.EventQueryOpts) ([]*database.ClusterEvent, error) {
	return s.history, nil
}
func (s contractQueryStub) CountEventHistory(context.Context, string) (int64, error) {
	return int64(len(s.history)), nil
}

func get(t *testing.T, h http.HandlerFunc, target string) []byte {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, target, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status = %d: %s", target, rec.Code, rec.Body)
	}
	return rec.Body.Bytes()
}

// withOverviewEnums 枚举字段换成规格允许的值（Fill 给普通字符串填的是 "x"）
func withOverviewEnums(o *cluster.ClusterOverview) *cluster.ClusterOverview {
	h := &o.Cards.ClusterHealth
	h.Status, h.NodeReadyStatus, h.PodReadyStatus = cluster.HealthDegraded, cluster.HealthHealthy, cluster.HealthDegraded
	if h.Reason != "" {
		h.Reason = cluster.HealthReasonPodReadyLow
	}
	for i := range o.Alerts.Recent {
		o.Alerts.Recent[i].Severity = "warning"
	}
	return o
}

func filledOverview() *cluster.ClusterOverview {
	var o cluster.ClusterOverview
	contract.Fill(&o, model.HardwareGood)
	return withOverviewEnums(&o)
}

// 集群概览：输入填满时每个属性都出现且合规；输入为空时仍合规（数组不是 null）
func TestOverview_MatchesSpec(t *testing.T) {
	const path = "/api/v2/overview"
	body := get(t, NewOverviewHandler(contractQueryStub{overview: filledOverview()}).Get, path+"?cluster_id=c1")
	contract.AssertResponse(t, http.MethodGet, path, http.StatusOK, body)

	body = get(t, NewOverviewHandler(contractQueryStub{overview: withOverviewEnums(&cluster.ClusterOverview{})}).Get, path+"?cluster_id=c1")
	contract.AssertValid(t, http.MethodGet, path, http.StatusOK, body)
}

// 事件列表（实时 / 历史两个来源同一形状）
func TestEventList_MatchesSpec(t *testing.T) {
	const path = "/api/v2/events"
	var ev cluster.Event
	contract.Fill(&ev)
	var row database.ClusterEvent
	contract.Fill(&row)
	h := NewEventHandler(contractQueryStub{events: []cluster.Event{ev}, history: []*database.ClusterEvent{&row}})

	contract.AssertResponse(t, http.MethodGet, path, http.StatusOK, get(t, h.List, path+"?cluster_id=c1&source=history&offset=1"))
	contract.AssertValid(t, http.MethodGet, path, http.StatusOK, get(t, h.List, path+"?cluster_id=c1"))

	empty := NewEventHandler(contractQueryStub{})
	contract.AssertValid(t, http.MethodGet, path, http.StatusOK, get(t, empty.List, path+"?cluster_id=c1"))
	contract.AssertValid(t, http.MethodGet, path, http.StatusOK, get(t, empty.List, path+"?cluster_id=c1&source=history"))
}
