package snapshot

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/testutil/mock"
)

// hangingOTelRepo 的查询一直等到 ctx 结束才返回
func hangingOTelRepo() *mock.OTelSummaryRepository {
	wait := func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	return &mock.OTelSummaryRepository{
		GetAPMSummaryFn: func(ctx context.Context) (int, int, float64, float64, float64, error) {
			return 0, 0, 0, 0, 0, wait(ctx)
		},
		GetSLOSummaryFn: func(ctx context.Context) (int, float64, error) { return 0, 0, wait(ctx) },
		GetMetricsSummaryFn: func(ctx context.Context) (int, float64, float64, float64, float64, error) {
			return 0, 0, 0, 0, 0, wait(ctx)
		},
	}
}

// K8s 快照不等 OTel：ClickHouse 卡住时 Collect 照样立刻产出快照（OTel 用最近一次刷新好的结果）
func TestCollect_DoesNotWaitForOTel(t *testing.T) {
	svc := newTestService(func(s *snapshotService) { s.otelSummaryRepo = hangingOTelRepo() })

	done := make(chan error, 1)
	go func() {
		_, err := svc.Collect(context.Background())
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Collect: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Collect 被卡住的 OTel 查询拖住了")
	}
}

// OTel 刷新受传入的 ctx 约束：取消（Agent 关停）后立刻返回
func TestRefreshOTel_StopsWhenContextCancelled(t *testing.T) {
	svc := newTestService(func(s *snapshotService) { s.otelSummaryRepo = hangingOTelRepo() })
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		svc.RefreshOTel(ctx)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("取消后 RefreshOTel 没有返回")
	}
}

// 刷新成功后，下一份快照带上它
func TestCollect_AttachesLatestOTel(t *testing.T) {
	svc := newTestService(func(s *snapshotService) {
		s.otelSummaryRepo = &mock.OTelSummaryRepository{
			GetAPMSummaryFn: func(context.Context) (int, int, float64, float64, float64, error) {
				return 3, 3, 1, 100, 10, nil
			},
			GetSLOSummaryFn: func(context.Context) (int, float64, error) { return 1, 1, nil },
			GetMetricsSummaryFn: func(context.Context) (int, float64, float64, float64, float64, error) {
				return 2, 10, 20, 30, 40, nil
			},
		}
	})
	svc.RefreshOTel(context.Background())
	snap, err := svc.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.OTel == nil || snap.OTel.TotalServices != 3 {
		t.Fatalf("快照应带上最近一次刷新的 OTel，实际 %+v", snap.OTel)
	}
}
