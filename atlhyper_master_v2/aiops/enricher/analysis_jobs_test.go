package enricher

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai"
)

// C：深度分析状态跟踪（内存，按事件记最近一次）
func TestAnalysisJobs_Lifecycle(t *testing.T) {
	now := time.Unix(1000, 0)
	j := newAnalysisJobs(func() time.Time { return now })

	if _, ok := j.get("inc-1"); ok {
		t.Fatal("未触发时应查不到")
	}
	if !j.start("inc-1") {
		t.Fatal("首次应可启动")
	}
	if j.start("inc-1") {
		t.Error("进行中不得重复启动")
	}
	st, _ := j.get("inc-1")
	if st.State != JobRunning || st.FinishedAt != nil {
		t.Errorf("应为 running：%+v", st)
	}

	now = now.Add(11 * time.Second)
	j.finish("inc-1", nil)
	st, _ = j.get("inc-1")
	if st.State != JobSucceeded || st.FinishedAt == nil || st.ErrorCode != "" {
		t.Errorf("应为 succeeded：%+v", st)
	}
	if !j.start("inc-1") {
		t.Error("结束后应可再次启动")
	}
}

func TestAnalysisJobs_FailureCarriesCode(t *testing.T) {
	j := newAnalysisJobs(time.Now)
	cases := map[error]string{
		fmt.Errorf("深度分析失败: AI 配置错误: %w", ai.ErrRoleNotAssigned): "ai_role_unassigned",
		fmt.Errorf("x: %w", context.DeadlineExceeded):                     "ai_timeout",
		errors.New("LLM 返回空响应"):                                          "ai_upstream",
	}
	for err, want := range cases {
		j.start("inc-x")
		j.finish("inc-x", err)
		st, _ := j.get("inc-x")
		if st.State != JobFailed || st.ErrorCode != want {
			t.Errorf("%v → %+v, want code %s", err, st, want)
		}
	}
}

// 旧记录不无限累积：结束超过保留期的条目在下次启动时清掉
func TestAnalysisJobs_PrunesOldFinished(t *testing.T) {
	now := time.Unix(1000, 0)
	j := newAnalysisJobs(func() time.Time { return now })
	j.start("old")
	j.finish("old", nil)
	now = now.Add(jobRetention + time.Minute)
	j.start("new")
	if _, ok := j.get("old"); ok {
		t.Error("超过保留期的已结束记录应被清理")
	}
}
