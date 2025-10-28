// atlhyper_aiservice/service/pipeline.go
package service

import (
	m "AtlHyper/model/event"
	"context"
)

// RunAIDiagnosisPipeline —— 统一的 AI 诊断流程入口
// 1️⃣ 调用 Stage1：AI 初步分析事件
// 2️⃣ 调用 Stage2：解析清单并请求 Master 上下文
// 3️⃣ 调用 Stage3：结合上下文再次调用 AI
func RunAIDiagnosisPipeline(ctx context.Context, clusterID string, events []m.EventLog) (map[string]interface{}, error) {
	// === ① 阶段一：AI 初步分析 ===
	stage1Resp, err := RunStage1Analysis(clusterID, events)
	if err != nil {
		return nil, err
	}

	// === ② 阶段二：解析清单并取 Master 上下文 ===
	stage2Resp, err := RunStage2FetchContext(ctx, clusterID, stage1Resp, events)
	if err != nil {
		return map[string]interface{}{
			"stage1": stage1Resp,
			"error":  err.Error(),
		}, nil
	}

	// === ③ 阶段三：最终分析 ===
	stage3Resp, err := RunStage3FinalDiagnosis(clusterID, stage1Resp, stage2Resp)
	if err != nil {
		return map[string]interface{}{
			"stage1": stage1Resp,
			"stage2": stage2Resp,
			"error":  err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"stage1": stage1Resp,
		"stage2": stage2Resp,
		"stage3": stage3Resp,
	}, nil
}
