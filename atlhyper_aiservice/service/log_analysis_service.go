// atlhyper_aiservice/service/log_analysis_service.go
package service

import (
	"AtlHyper/atlhyper_aiservice/client"
	"AtlHyper/atlhyper_aiservice/config"
	m "AtlHyper/model/event" // ✅ 使用统一的事件结构体 model.EventLog
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

//
// DiagnoseEvents —— 第一步：AI 初步分析集群事件
// ------------------------------------------------------------
// ✅ 职责说明：
//   - 对传入的 Kubernetes 事件进行分组、格式化
//   - 生成结构化 Prompt 以便 AI 理解（包含 Severity 层级）
//   - 调用 Gemini 模型进行文本分析，推断潜在根因与建议
//
// ✅ 输入：
//   - clusterID：当前集群 ID
//   - events：来自 master 的事件列表（[]model.EventLog）
//
// ✅ 输出：
//   - map[string]interface{}：包含分析摘要、prompt 原文、AI 输出
//
// ⚙️ 调用链：
//   Master → POST /ai/diagnose → DiagnoseEventHandler → DiagnoseEvents()
//
// 🚀 分析流程：
//   1. 按严重程度 (Severity) 对事件分组
//   2. 生成清晰的上下文 prompt（让 AI 能看到结构化信息）
//   3. 调用 Gemini 模型进行自然语言推理
//   4. 返回分析摘要与原始输出（便于日志与后续判断）
//
func DiagnoseEvents(clusterID string, events []m.EventLog) (map[string]interface{}, error) {
	// 基础校验
	if len(events) == 0 {
		return nil, fmt.Errorf("no events to analyze")
	}

	// -------------------------------------------------------------------
	// 1️⃣ 按事件严重性分组
	// -------------------------------------------------------------------
	// 目的：帮助 AI 更容易区分“高危”“警告”“信息”等类型的事件，
	//       从而更精确地分析潜在根因。
	// 例：
	//   Critical: [节点宕机]
	//   Warning:  [Pod CrashLoopBackOff]
	//   Info:     [Deployment Scaling]
	grouped := map[string][]m.EventLog{}
	for _, e := range events {
		key := e.Severity
		if key == "" {
			key = "Unknown" // 若无严重级别，则归为 Unknown
		}
		grouped[key] = append(grouped[key], e)
	}

	// 固定排序，保证输出稳定（避免 map 随机顺序）
	severities := make([]string, 0, len(grouped))
	for sev := range grouped {
		severities = append(severities, sev)
	}
	sort.Strings(severities)

	// -------------------------------------------------------------------
	// 2️⃣ 构造 Prompt —— 让 AI 理解上下文
	// -------------------------------------------------------------------
	// 样例格式：
	//   集群 ID: cluster-prod
	//   以下是最新检测到的 Kubernetes 事件（按严重性分组）：
	//
	//   【Critical 级事件】
	//   - [Node] desk-one → MemoryPressure: node memory low
	//     ↳ Message: Node stability issue (Time: 2025-10-28T09:00:00Z)
	//
	//   【Warning 级事件】
	//   - [Pod] media/desk-one → CrashLoopBackOff: container restart
	//     ↳ Message: Pod restarted 5 times (Time: 2025-10-28T09:02:00Z)
	//
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("集群 ID: %s\n", clusterID))
	sb.WriteString("以下是最新检测到的 Kubernetes 事件（按严重性分组）：\n\n")

	for _, sev := range severities {
		sb.WriteString(fmt.Sprintf("【%s 级事件】\n", sev))
		for _, e := range grouped[sev] {
			sb.WriteString(fmt.Sprintf(
				"- [%s] %s/%s → %s: %s\n  ↳ Message: %s (Time: %s)\n",
				e.Kind, e.Namespace, e.Node, e.Reason, e.Message, e.Category, e.EventTime))
		}
		sb.WriteString("\n")
	}

	// 为 AI 提供明确的任务说明（Prompt 指令）
	sb.WriteString("请你：\n")
	sb.WriteString("1. 结合以上事件，分析潜在根因与可能的影响范围。\n")
	sb.WriteString("2. 指出建议进一步分析的资源类型（如 node、namespace、deployment、service 等）。\n")
	sb.WriteString("3. 如果事件相互关联，请推测可能的关联路径（例如：Node 故障 → Pod 崩溃 → Service 不可用）。\n")

	prompt := sb.String()

	// -------------------------------------------------------------------
	// 3️⃣ 调用 Gemini 进行 AI 分析
	// -------------------------------------------------------------------
	cfg := config.GetGeminiConfig()
	ctx := context.Background()
	c, err := client.GetGeminiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Gemini 客户端失败: %v", err)
	}

	model := c.GenerativeModel(cfg.ModelName)

	// 向 Gemini 发送 prompt 请求
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("AI 分析失败: %v", err)
	}
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("AI 无响应")
	}

	// -------------------------------------------------------------------
	// 4️⃣ 提取输出结果
	// -------------------------------------------------------------------
	// Gemini 可能返回多段内容（Parts），这里拼接为完整输出。
	out := ""
	for _, p := range resp.Candidates[0].Content.Parts {
		out += fmt.Sprintf("%v", p)
	}

	// -------------------------------------------------------------------
	// 5️⃣ 返回结构
	// -------------------------------------------------------------------
	// summary: 任务完成摘要（供日志或前端展示）
	// prompt:  发送给 AI 的原始文本（调试用）
	// ai_raw:  AI 的完整输出（可能包含诊断结论或下一步建议）
	return map[string]interface{}{
		"summary": fmt.Sprintf("✅ 初步分析完成（cluster=%s）", clusterID),
		"prompt":  prompt,
		"ai_raw":  out,
	}, nil
}
