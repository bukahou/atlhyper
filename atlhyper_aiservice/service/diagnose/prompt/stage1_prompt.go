package prompt

import (
	m "AtlHyper/model/event"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// BuildStage1Prompt —— 阶段1提示词构造：基于事件日志生成AI输入
// ------------------------------------------------------------
// 输入：clusterID + EventLog 列表
// 输出：完整 Prompt 字符串（供 Gemini / LLM 使用）
func BuildStage1Prompt(clusterID string, events []m.EventLog) string {
	// Step1️⃣: 分组
	grouped := map[string][]m.EventLog{}
	for _, e := range events {
		key := e.Severity
		if key == "" {
			key = "Unknown"
		}
		grouped[key] = append(grouped[key], e)
	}

	// Step2️⃣: 排序
	sevs := make([]string, 0, len(grouped))
	for k := range grouped {
		sevs = append(sevs, k)
	}
	sort.Strings(sevs)

	// Step3️⃣: 构造 Prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("集群 ID: %s\n\n", clusterID))
	sb.WriteString("以下为结构化的 Kubernetes 事件数据（JSON 格式）：\n")
	sb.WriteString("字段说明：ClusterID, Kind, Namespace, Name, Node, Severity, Reason, Message\n\n")

	jb, _ := json.MarshalIndent(events, "", "  ")
	sb.WriteString(string(jb))
	sb.WriteString("\n\n")

	sb.WriteString(`请特别注意：
1. Name 必须结合 Kind 判断资源类型。
2. 不得虚构或修改资源。
3. 若不存在某类资源，请输出空数组 []。
4. clusterID 必须为 ` + clusterID + `。

请基于以上事件输出以下结构化 JSON：
{
  "summary": "简述整体事件现象",
  "rootCause": "根本原因分析",
  "impact": "影响范围说明",
  "recommendation": "修复与排查建议",
  "needResources": {
    "clusterID": "` + clusterID + `",
    "pods": [],
    "deployments": [],
    "services": [],
    "nodes": [],
    "configMaps": [],
    "namespaces": [],
    "ingresses": [],
    "endpointSlices": []
  }
}
`)

	return sb.String()
}
