package aiops

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops/enricher"
)

type fakeTrigger struct {
	running map[string]bool
	jobs    map[string]enricher.AnalysisJob
}

func (f *fakeTrigger) TriggerAnalysis(id, _ string) bool {
	if f.running[id] {
		return false
	}
	f.running[id] = true
	return true
}

func (f *fakeTrigger) AnalysisStatus(id string) (enricher.AnalysisJob, bool) {
	j, ok := f.jobs[id]
	return j, ok
}

func newTriggerHandler(f *fakeTrigger) *AIOpsAIHandler {
	h := NewAIOpsAIHandler(nil)
	h.SetAnalyzeTrigger(f)
	return h
}

// 重复触发：不再启动第二个，告诉前端「已在进行」，前端照常轮询状态
func TestAnalyzeHandler_AlreadyRunning(t *testing.T) {
	f := &fakeTrigger{running: map[string]bool{}, jobs: map[string]enricher.AnalysisJob{}}
	h := newTriggerHandler(f)
	call := func() map[string]any {
		rec := httptest.NewRecorder()
		h.AnalyzeHandler(rec, httptest.NewRequest(http.MethodPost, "/api/v2/aiops/ai/analyze", strings.NewReader(`{"incidentId":"inc-1"}`)))
		var body map[string]any
		json.Unmarshal(rec.Body.Bytes(), &body)
		return body
	}
	if call()["alreadyRunning"] != false {
		t.Error("首次触发 alreadyRunning 应为 false")
	}
	if call()["alreadyRunning"] != true {
		t.Error("进行中再次触发 alreadyRunning 应为 true")
	}
}

func TestAnalyzeStatus(t *testing.T) {
	fin := time.Unix(2000, 0)
	f := &fakeTrigger{running: map[string]bool{}, jobs: map[string]enricher.AnalysisJob{
		"inc-run":  {IncidentID: "inc-run", State: enricher.JobRunning, StartedAt: time.Unix(1000, 0)},
		"inc-fail": {IncidentID: "inc-fail", State: enricher.JobFailed, ErrorCode: "ai_role_unassigned", StartedAt: time.Unix(1000, 0), FinishedAt: &fin},
	}}
	h := newTriggerHandler(f)
	get := func(id string) map[string]any {
		rec := httptest.NewRecorder()
		h.AnalyzeStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v2/aiops/ai/analyze/status?incidentId="+id, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: code %d", id, rec.Code)
		}
		var body struct {
			Data map[string]any `json:"data"`
		}
		json.Unmarshal(rec.Body.Bytes(), &body)
		return body.Data
	}
	if d := get("inc-run"); d["state"] != "running" {
		t.Errorf("running: %v", d)
	}
	if d := get("inc-fail"); d["state"] != "failed" || d["errorCode"] != "ai_role_unassigned" {
		t.Errorf("failed: %v", d)
	}
	// 查不到（未触发过 / Master 重启后丢失）→ none
	if d := get("inc-unknown"); d["state"] != "none" {
		t.Errorf("unknown: %v", d)
	}
}
