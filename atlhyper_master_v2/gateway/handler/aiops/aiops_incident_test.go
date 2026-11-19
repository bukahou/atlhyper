package aiops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/service"
)

// stubIncidentQuery 只实现事件列表与详情两个查询
type stubIncidentQuery struct {
	service.Query
	incidents []*aiops.Incident
	detail    *aiops.IncidentDetail
}

func (s *stubIncidentQuery) GetAIOpsIncidents(context.Context, aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error) {
	return s.incidents, len(s.incidents), nil
}

func (s *stubIncidentQuery) GetAIOpsIncidentDetail(context.Context, string) (*aiops.IncidentDetail, error) {
	return s.detail, nil
}

func closedWorkloadIncident() aiops.Incident {
	startedAt := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	escalatedAt := startedAt.Add(7 * time.Minute)
	resolvedAt := startedAt.Add(50 * time.Hour)
	return aiops.Incident{
		ID: "inc-1", ClusterID: "c1", State: aiops.StateStable, Severity: "high",
		Subject: "shop/deployment/api", SubjectType: "deployment", RootCause: "shop/pod/api-a",
		PeakRisk: 0.75, StartedAt: startedAt, EscalatedAt: &escalatedAt, ResolvedAt: &resolvedAt,
		CloseReason: aiops.CloseReasonRecovered, Recurrence: 2,
	}
}

func serveJSON(t *testing.T, handle http.HandlerFunc, target string) map[string]any {
	t.Helper()
	recorder := httptest.NewRecorder()
	handle(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s = %d: %s", target, recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v", err)
	}
	return body
}

// assertSubjectFields 事件的主体、升级时间、关闭原因以 camelCase 出现在响应里
func assertSubjectFields(t *testing.T, incident map[string]any) {
	t.Helper()
	want := map[string]any{
		"subject":     "shop/deployment/api",
		"subjectType": "deployment",
		"escalatedAt": "2026-01-01T08:07:00Z",
		"closeReason": "recovered",
		"severity":    "high", // 已关闭的事件仍带着峰值严重度
		"rootCause":   "shop/pod/api-a",
		"recurrence":  float64(2),
	}
	for key, value := range want {
		if got, ok := incident[key]; !ok || got != value {
			t.Errorf("%s = %v（存在=%v），want %v", key, got, ok, value)
		}
	}
}

func TestIncidentList_ExposesSubjectFields(t *testing.T) {
	inc := closedWorkloadIncident()
	h := NewAIOpsIncidentHandler(&stubIncidentQuery{incidents: []*aiops.Incident{&inc}})

	body := serveJSON(t, h.List, "/api/v2/aiops/incidents?cluster_id=c1")

	data, ok := body["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data 应有 1 条事件，实际 %v", body["data"])
	}
	assertSubjectFields(t, data[0].(map[string]any))
}

func TestIncidentDetail_ExposesSubjectFields(t *testing.T) {
	detail := &aiops.IncidentDetail{
		Incident: closedWorkloadIncident(),
		Entities: []*aiops.IncidentEntity{
			{IncidentID: "inc-1", EntityKey: "shop/pod/api-a", EntityType: "pod", RLocal: 0.8, RFinal: 0.75, Role: "root_cause"},
			{IncidentID: "inc-1", EntityKey: "shop/pod/api-b", EntityType: "pod", RLocal: 0.3, RFinal: 0.3, Role: "affected"},
		},
	}
	h := NewAIOpsIncidentHandler(&stubIncidentQuery{detail: detail})

	body := serveJSON(t, h.Detail, "/api/v2/aiops/incidents/inc-1")

	assertSubjectFields(t, body)
	// 风险值的刻度约定不变：详情里是 [0, 100]
	if body["peakRisk"] != float64(75) {
		t.Errorf("peakRisk = %v, want 75", body["peakRisk"])
	}
	entities, _ := body["entities"].([]any)
	if len(entities) != 2 || entities[0].(map[string]any)["rFinal"] != float64(75) {
		t.Errorf("成员实体应带出且风险为 [0, 100] 刻度: %v", body["entities"])
	}
}

// 从未升级、尚未关闭的事件：escalatedAt 为 null、closeReason 为空串（字段始终存在）
func TestIncidentList_OpenIncidentHasNullEscalatedAt(t *testing.T) {
	inc := aiops.Incident{ID: "inc-2", ClusterID: "c1", State: aiops.StateWarning, Severity: "low",
		Subject: "shop/service/api", SubjectType: "service", RootCause: "shop/service/api"}
	h := NewAIOpsIncidentHandler(&stubIncidentQuery{incidents: []*aiops.Incident{&inc}})

	body := serveJSON(t, h.List, "/api/v2/aiops/incidents?cluster_id=c1")

	got := body["data"].([]any)[0].(map[string]any)
	if value, ok := got["escalatedAt"]; !ok || value != nil {
		t.Errorf("escalatedAt 应存在且为 null，实际 %v（存在=%v）", value, ok)
	}
	if value, ok := got["closeReason"]; !ok || value != "" {
		t.Errorf("closeReason 应存在且为空串，实际 %v（存在=%v）", value, ok)
	}
}

// 列表与事件模式里的峰值风险和详情用同一个 0-100 刻度
func TestScaleIncidents_PeakRiskToPercent(t *testing.T) {
	src := []*aiops.Incident{{ID: "inc-1", PeakRisk: 0.75}}

	scaled := ScaleIncidents(src)
	if scaled[0].PeakRisk != 75 {
		t.Errorf("列表的 peakRisk = %v，期望 75", scaled[0].PeakRisk)
	}
	if src[0].PeakRisk != 0.75 {
		t.Errorf("换算不应改动原对象，实际 %v", src[0].PeakRisk)
	}

	patterns := ScaleIncidentPatterns([]*aiops.IncidentPattern{{EntityKey: "shop/deployment/api", Incidents: src}})
	if patterns[0].Incidents[0].PeakRisk != 75 {
		t.Errorf("事件模式的 peakRisk = %v，期望 75", patterns[0].Incidents[0].PeakRisk)
	}
}
