package service

import (
	model "AtlHyper/model/ai"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// RunStage2ParseNeedResources —— 阶段2a：解析 AI 输出中的 needResources 字段
// --------------------------------------------------------------
// 📘 功能：
//   1️⃣ 从 Stage1 的 ai_json / ai_raw 中提取 needResources。
//   2️⃣ 清理 ```json 包裹。
//   3️⃣ 转换为 model.AIFetchRequest 结构体。
//   4️⃣ 返回 req，用于下一个阶段调用 Master。
func RunStage2ParseNeedResources(ctx context.Context, clusterID string, stage1 map[string]interface{}) (*model.AIFetchRequest, error) {
	var req model.AIFetchRequest
	req.ClusterID = clusterID

	var rawJSON string

	// Step 1️⃣ 优先从 ai_json.needResources 获取
	if j, ok := stage1["ai_json"].(map[string]interface{}); ok {
		if nr, ok := j["needResources"]; ok {
			if b, err := json.Marshal(nr); err == nil {
				_ = json.Unmarshal(b, &req)
				return &req, nil
			}
		}
		// Step 2️⃣ 如果 ai_json.raw 存在，说明模型输出是字符串化 JSON
		if raw, ok := j["raw"].(string); ok {
			rawJSON = raw
		}
	}

	// Step 3️⃣ 或者直接从 ai_raw 中获取
	if rawJSON == "" {
		if raw, ok := stage1["ai_raw"].(string); ok {
			rawJSON = raw
		}
	}
	if rawJSON == "" {
		return &req, fmt.Errorf("no valid AI JSON found in stage1 output")
	}

	// Step 4️⃣ 清理 Markdown 包裹 ```json ... ```
	out := strings.TrimSpace(rawJSON)
	re := regexp.MustCompile("(?s)^```json\\s*(.*?)\\s*```$")
	if matches := re.FindStringSubmatch(out); len(matches) > 1 {
		out = matches[1]
	}

	// Step 5️⃣ 尝试解析 needResources
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err == nil {
		if nr, ok := parsed["needResources"]; ok {
			b, _ := json.Marshal(nr)
			_ = json.Unmarshal(b, &req)
			return &req, nil
		}
	}

	// Step 6️⃣ 若失败则尝试从 { 开始再解析一次
	if idx := strings.Index(out, "{"); idx != -1 {
		if err := json.Unmarshal([]byte(out[idx:]), &parsed); err == nil {
			if nr, ok := parsed["needResources"]; ok {
				b, _ := json.Marshal(nr)
				_ = json.Unmarshal(b, &req)
				return &req, nil
			}
		}
	}

	return &req, fmt.Errorf("failed to extract needResources from AI output")
}
