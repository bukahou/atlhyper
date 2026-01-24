// atlhyper_master_v2/ai/prompt.go
// 提示词构建
// 使用 go:embed 加载提示词文件，拼接为完整的 system prompt
package ai

import (
	"embed"
	"encoding/json"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

//go:embed prompts/security.txt
var securityPrompt string

//go:embed prompts/role.txt
var rolePrompt string

//go:embed prompts/tools.json
var toolsJSON string

// BuildSystemPrompt 构建系统提示词
// L0(security) + L1(role) 拼接
func BuildSystemPrompt() string {
	return securityPrompt + "\n\n" + rolePrompt
}

// LoadToolDefinitions 加载 Tool 定义
// 从 prompts/tools.json 解析为 llm.ToolDefinition 列表
func LoadToolDefinitions() ([]llm.ToolDefinition, error) {
	var rawTools []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	}
	if err := json.Unmarshal([]byte(toolsJSON), &rawTools); err != nil {
		return nil, err
	}

	tools := make([]llm.ToolDefinition, len(rawTools))
	for i, t := range rawTools {
		tools[i] = llm.ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		}
	}
	return tools, nil
}

// toolsCache 缓存加载的 Tool 定义
var toolsCache []llm.ToolDefinition

// GetToolDefinitions 获取 Tool 定义（带缓存）
func GetToolDefinitions() []llm.ToolDefinition {
	if toolsCache == nil {
		var err error
		toolsCache, err = LoadToolDefinitions()
		if err != nil {
			return nil
		}
	}
	return toolsCache
}

// 确保 embed 包被使用
var _ embed.FS
