// atlhyper_aiservice/service/diagnose/pipeline.go
package diagnose

import (
	"context"
	"encoding/json"
	"fmt"

	m "AtlHyper/model/event"
)

// RunAIDiagnosisPipeline —— 统一的 AI 诊断流程
// --------------------------------------------------------------
// 1️⃣ Stage1: 初步分析
// 2️⃣ Stage2a: 解析 needResources
// 3️⃣ Stage2b: 向 Master 请求上下文
// 4️⃣ Stage3: 最终 AI 诊断
func RunAIDiagnosisPipeline(ctx context.Context, clusterID string, events []m.EventLog) (map[string]interface{}, error) {

	fmt.Printf("\n==============================\n")
	fmt.Printf("🚀 [Pipeline] 启动 AI 诊断流程 (ClusterID: %s)\n", clusterID)
	fmt.Printf("📦 输入事件数量: %d\n", len(events))
	fmt.Printf("==============================\n\n")

	// === ① Stage1: AI 初步分析 ===
	fmt.Println("🧠 [Stage1] 开始初步分析...")
	stage1Resp, err := RunStage1Analysis(clusterID, events)
	if err != nil {
		fmt.Printf("❌ [Stage1] 分析失败: %v\n", err)
		return nil, err
	}
	fmt.Println("✅ [Stage1] 完成初步分析！")
	if summary, ok := stage1Resp["summary"]; ok {
		fmt.Printf("   └─ 摘要: %v\n", summary)
	}
	fmt.Println()

	// === ② Stage2a: 解析 needResources ===
	fmt.Println("🔍 [Stage2a] 解析 AI 输出中的 needResources 清单...")
	req, err := RunStage2ParseNeedResources(ctx, clusterID, stage1Resp)
	if err != nil {
		fmt.Printf("❌ [Stage2a] 解析失败: %v\n", err)
		return map[string]interface{}{
			"stage1": stage1Resp,
			"error":  "解析 AI 输出失败: " + err.Error(),
		}, nil
	}
	fmt.Println("✅ [Stage2a] 成功解析 needResources 清单！")
	b, _ := json.MarshalIndent(req, "   ", "  ")
	fmt.Printf("   └─ 解析结果:\n%s\n\n", string(b))
	// fmt.Println("🎯 [Pipeline] 在 Stage2a 结束（调试模式：仅输出解析结果）")
	// return map[string]interface{}{
	// 	"stage1": stage1Resp,
	// 	"stage2a": req,
	// }, nil


	// === ③ Stage2b: 获取上下文 ===
	fmt.Println("🌐 [Stage2b] 请求 Master 获取上下文数据...")
	stage2Resp, err := RunStage3FetchContext(ctx, req)
	if err != nil {
		fmt.Printf("❌ [Stage2b] 上下文获取失败: %v\n", err)
		return map[string]interface{}{
			"stage1": stage1Resp,
			"stage2": req,
			"error":  "上下文获取失败: " + err.Error(),
		}, nil
	}
	fmt.Println("✅ [Stage2b] 成功从 Master 获取上下文数据！")
	if fetch, ok := stage2Resp["fetch"]; ok {
		fmt.Printf("   └─ 上下文数据大小约: %.2f KB\n\n", float64(len(fmt.Sprintf("%v", fetch)))/1024)
	}

	// === ④ Stage3: 最终分析 ===
	fmt.Println("🎯 [Stage3] 开始最终诊断分析...")
	needOnly := map[string]interface{}{"needResources": req}
	stage3Resp, err := RunStage3FinalDiagnosis(clusterID, stage1Resp, needOnly)
	if err != nil {
		fmt.Printf("❌ [Stage3] 诊断失败: %v\n", err)
		return map[string]interface{}{
			"stage1": stage1Resp,
			"stage2": stage2Resp,
			"error":  "最终诊断失败: " + err.Error(),
		}, nil
	}
	fmt.Println("✅ [Stage3] 最终 AI 诊断完成！")
	fmt.Println("---------------------------------------------------")
	if summary, ok := stage3Resp["finalSummary"]; ok {
		fmt.Printf("🧾 最终结论摘要: %v\n", summary)
	}
	fmt.Println("---------------------------------------------------")

	fmt.Println("🎉 [Pipeline] 全流程执行完毕！")

	return map[string]interface{}{
		"stage1": stage1Resp,
		"stage2": stage2Resp,
		"stage3": stage3Resp,
	}, nil
}
