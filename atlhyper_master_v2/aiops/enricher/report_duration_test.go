package enricher

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/database"
)

type captureReportRepo struct {
	database.AIReportRepository
	saved []*database.AIReport
}

func (c *captureReportRepo) Create(_ context.Context, r *database.AIReport) error {
	c.saved = append(c.saved, r)
	return nil
}

// durationAIService 在 mockAIService 基础上让 Complete / Analyze 带回耗时
type durationAIService struct {
	*mockAIService
}

func (d *durationAIService) Complete(ctx context.Context, req *ai.CompleteRequest) (*ai.CompleteResult, error) {
	r, err := d.mockAIService.Complete(ctx, req)
	if r != nil {
		r.Duration = 1500 * time.Millisecond
	}
	return r, err
}

func (d *durationAIService) Analyze(context.Context, *ai.AnalyzeRequest) (*ai.AnalyzeResult, error) {
	return &ai.AnalyzeResult{Response: `{"summary":"s","rootCauseAnalysis":"r","recommendations":[]}`,
		ProviderName: "p", Model: "m", Duration: 11 * time.Second}, nil
}

// A6：报告耗时曾恒为 0.0s（保存报告时没写 DurationMs）
func TestReports_RecordLLMDuration(t *testing.T) {
	repo := &mockIncidentRepo{incident: makeTestIncident(), entities: makeTestEntities(), timeline: makeTestTimeline()}
	reports := &captureReportRepo{}
	svc := &durationAIService{newMockAIService(`{"summary":"s","rootCauseAnalysis":"r","recommendations":[],"similarPattern":"无"}`)}
	e := NewEnricher(repo, reports, svc)

	if _, err := e.SummarizeBackground(context.Background(), "inc-test-001", "manual"); err != nil {
		t.Fatal(err)
	}
	if err := e.runAnalysis(context.Background(), "inc-test-001", "manual"); err != nil {
		t.Fatal(err)
	}
	if len(reports.saved) != 2 {
		t.Fatalf("应保存 2 份报告，得到 %d", len(reports.saved))
	}
	got := map[string]int64{}
	for _, r := range reports.saved {
		got[r.Role] = r.DurationMs
	}
	if got["background"] != 1500 || got["analysis"] != 11000 {
		t.Errorf("耗时未写入报告：%v", got)
	}
}
