package service

import (
	"AtlHyper/atlhyper_aiservice/client/ai"
	"AtlHyper/atlhyper_aiservice/config"
	m "AtlHyper/model/event"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

//
// RunStage1Analysis —— 执行 AI 诊断的第一阶段：事件初步分析
// ----------------------------------------------------------------------
// 📘 功能说明：
//   该函数接收来自 Master 的事件列表，调用 Gemini 模型进行初步分析，
//   自动生成聚合报告（summary / rootCause / impact / recommendation 等）。
//
// 🔧 逻辑步骤：
//   1. 构造输入 Prompt（按事件严重性分组，提供上下文信息）
//   2. 调用 Gemini API 执行自然语言分析
//   3. 尝试解析返回内容为 JSON（若失败则保留原始文本）
//   4. 统一返回结构体，包含原始输入、AI 输出与摘要说明
//
// 🧩 参数说明：
//   - clusterID：集群唯一标识符（例如 cluster-1）
//   - events：事件列表（来自 model/event.EventLog）
//
// 📤 返回值说明：
//   - map[string]interface{}：包含分析结果的通用结构体：
//       {
//         "summary": "✅ 初步分析完成（cluster=xxx）",
//         "prompt": "AI 输入 Prompt 内容",
//         "ai_json": {summary, rootCause, impact, recommendation, needResources},
//         "ai_raw": "AI 原始输出"
//       }
//   - error：出现调用或解析错误时返回错误信息
//
func RunStage1Analysis(clusterID string, events []m.EventLog) (map[string]interface{}, error) {
	// 🧭 Step 1. 参数检查
	if len(events) == 0 {
		return nil, fmt.Errorf("no events to analyze") // 没有事件可供分析
	}

	// 🧠 Step 2. 构造 AI Prompt 输入
	prompt := buildStage1Prompt(clusterID, events)

	// ⚙️ Step 3. 初始化 Gemini 客户端
	cfg := config.GetGeminiConfig()             // 获取模型配置（ModelName / APIKey）
	ctx := context.Background()                 // 创建上下文
	c, err := ai.GetGeminiClient(ctx)           // 获取 Gemini API 客户端
	if err != nil {
		return nil, fmt.Errorf("get gemini client failed: %v", err)
	}
	model := c.GenerativeModel(cfg.ModelName)   // 选择模型（如 gemini-2.5-flash）

	// 🚀 Step 4. 调用 AI 模型执行分析
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %v", err)
	}

	// 🪄 Step 5. 拼接 AI 原始输出（Gemini 返回内容以多段形式存在）
	out := ""
	for _, p := range resp.Candidates[0].Content.Parts {
		out += fmt.Sprintf("%v", p)
	}

	// 🧩 Step 6. 尝试解析输出为 JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		// 若首部存在多余文本，则尝试从 “{” 开始重新解析
		if idx := strings.Index(out, "{"); idx != -1 {
			_ = json.Unmarshal([]byte(out[idx:]), &parsed)
		}
	}

	// 🧱 Step 7. 若无法解析出结构化 JSON，则保留原始文本
	if parsed == nil {
		parsed = map[string]interface{}{"raw": out}
	}

	// 🧾 Step 8. 构造统一返回结果
	return map[string]interface{}{
		"summary": fmt.Sprintf("✅ 初步分析完成（cluster=%s）", clusterID),
		"prompt":  prompt,  // 输入提示词内容
		"ai_json": parsed,  // 解析后的 AI JSON 输出
		"ai_raw":  out,     // 原始文本输出
	}, nil
}

//
// buildStage1Prompt —— 构造用于 AI 分析的 Prompt 内容
// ----------------------------------------------------------------------
// 📘 功能说明：
//   将事件列表按严重性（Severity）分组，并格式化成自然语言描述，
//   为 Gemini 模型提供可理解的上下文输入。
//
// 🧩 参数说明：
//   - clusterID：集群唯一标识符
//   - events：事件日志列表
//
// 📤 返回值说明：
//   - string：构造完成的 Prompt 文本
//
// 🧠 Prompt 内容包含：
//   1. 集群 ID
//   2. 按严重性分组的事件明细
//   3. 指定 AI 输出格式（必须是 JSON）
//
func buildStage1Prompt(clusterID string, events []m.EventLog) string {
	// 🧭 Step 1. 按事件严重性分组
	grouped := map[string][]m.EventLog{}
	for _, e := range events {
		key := e.Severity
		if key == "" {
			key = "Unknown"
		}
		grouped[key] = append(grouped[key], e)
	}

	// 🧾 Step 2. 排序（保证稳定输出）
	sevs := make([]string, 0, len(grouped))
	for k := range grouped {
		sevs = append(sevs, k)
	}
	sort.Strings(sevs)

	// 🧱 Step 3. 构造 Prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("集群 ID: %s\n\n", clusterID))
	sb.WriteString("以下是结构化的 Kubernetes 事件数据（JSON 格式）：\n")
	sb.WriteString("每个事件对象包含以下字段：\n")
	sb.WriteString(`
- ClusterID：事件所属集群。
- Kind：资源类型（Pod / Deployment / Service / EndpointSlice / Node 等）。
- Namespace：资源命名空间。
- Name：资源名称（必须结合 Kind 一起识别资源类型）。
- Node：资源运行的节点（仅 Pod 类型有效）。
- Severity：事件严重级别（critical / warning / info / 等）。
- Reason：事件原因（例如 CrashLoopBackOff / UnavailableReplica）。
- Message：事件内容或描述。
`)

	sb.WriteString("\n以下为实际事件 JSON 数据，请直接读取字段值，不要进行语义推测：\n")
	jb, _ := json.MarshalIndent(events, "", "  ")
	sb.WriteString(string(jb))
	sb.WriteString("\n\n")

	sb.WriteString(`请特别注意：
1. Name 字段必须结合 Kind 来判断资源类型。
   - 若 Kind 为 "Pod"，则该对象属于 needResources.pods。
   - 若 Kind 为 "Deployment"，则属于 needResources.deployments。
   - 若 Kind 为 "Service"，则属于 needResources.services。
   - 若 Kind 为 "EndpointSlice"，则属于 needResources.endpointSlices。
   - 若 Kind 为 "Node"，则属于 needResources.nodes。
2. 不要凭空创建或修改资源名。
3. 若事件中不存在某类资源，请输出空数组 []。
4. 所有提取的命名空间、名称、节点名都必须与上方 JSON 完全一致。
5. clusterID 必须等于输入的集群 ID（` + clusterID + `）。

`)

	sb.WriteString(`
请你基于这些事件，分析问题的现象、根因、影响与建议，
并输出严格遵循以下 JSON 结构的结果。

输出要求：
- 必须输出合法 JSON，禁止添加注释或说明文字。
- 字段名、层级、类型必须完全一致。
- 所有资源引用必须从上方 JSON 的字段中提取。

JSON 输出模板：
{
  "summary": "string —— 对事件总体现象的简要描述。",
  "rootCause": "string —— 可能的根本原因说明。",
  "impact": "string —— 事件影响范围，例如影响到哪些服务或节点。",
  "recommendation": "string —— 针对本次事件的修复或排查建议。",
  "needResources": {
    "clusterID": "string —— 必须与输入 clusterID 一致。",
    "pods": [{"namespace": "string", "name": "string"}],
    "deployments": [{"namespace": "string", "name": "string"}],
    "services": [{"namespace": "string", "name": "string"}],
    "nodes": ["string"],
    "configMaps": [{"namespace": "string", "name": "string"}],
    "namespaces": [{"namespace": "string", "name": "string"}],
    "ingresses": [{"namespace": "string", "name": "string"}],
    "endpointSlices": [{"namespace": "string", "name": "string"}]
  }
}
`)

	return sb.String()
}
