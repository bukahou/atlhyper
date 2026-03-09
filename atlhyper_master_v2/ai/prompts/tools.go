// atlhyper_master_v2/ai/prompts/tools.go
// Tool JSON 定义 + 加载/缓存函数
package prompts

import (
	"encoding/json"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// toolsJSON Tool 定义 JSON — 单一通用 Tool
const toolsJSON = `[
  {
    "name": "query_cluster",
    "description": "查询 Kubernetes 集群数据（只读）。通过 API Server 直连获取任意资源信息。可在一次回复中多次调用以并行获取数据。",
    "parameters": {
      "type": "object",
      "properties": {
        "action": {
          "type": "string",
          "description": "操作类型: get, list, describe, get_logs, get_events, get_configmap",
          "enum": ["get", "list", "describe", "get_logs", "get_events", "get_configmap"]
        },
        "kind": {
          "type": "string",
          "description": "Kubernetes 资源类型: Pod, Deployment, Service, Node, HPA, StatefulSet, DaemonSet, Job, CronJob, PVC, PV, Ingress, ConfigMap, NetworkPolicy, ReplicaSet, Endpoints, Namespace, ServiceAccount, Event 等"
        },
        "namespace": {
          "type": "string",
          "description": "命名空间。list 操作时可不填以查询所有命名空间；get/describe 需要填写。集群级资源（Node、PV、Namespace）始终不需要填写"
        },
        "name": {
          "type": "string",
          "description": "资源名称（list 操作时可不填）"
        },
        "label_selector": {
          "type": "string",
          "description": "标签选择器，如 app=nginx（list 时可用，用于过滤）"
        },
        "container": {
          "type": "string",
          "description": "容器名称（get_logs 时多容器 Pod 需要指定）"
        },
        "tail_lines": {
          "type": "integer",
          "description": "返回日志的尾行数（get_logs 时使用，默认 100）"
        },
        "involved_kind": {
          "type": "string",
          "description": "关联资源类型（get_events 时过滤用，如 Pod、Node、Deployment）"
        },
        "involved_name": {
          "type": "string",
          "description": "关联资源名称（get_events 时过滤用）"
        }
      },
      "required": ["action", "kind"]
    }
  },
  {
    "name": "analyze_incident",
    "description": "分析指定事件的根因、影响面和处置建议。输入事件 ID，返回 AI 分析结果。",
    "parameters": {
      "type": "object",
      "properties": {
        "incident_id": {
          "type": "string",
          "description": "事件 ID，格式如 inc-1737364200"
        }
      },
      "required": ["incident_id"]
    }
  },
  {
    "name": "get_cluster_risk",
    "description": "获取集群当前的风险评分和高风险实体。返回 ClusterRisk 分数 (0-100) 和 Top N 风险实体列表。",
    "parameters": {
      "type": "object",
      "properties": {
        "top_n": {
          "type": "integer",
          "description": "返回前 N 个高风险实体，默认 10"
        }
      }
    }
  },
  {
    "name": "get_recent_incidents",
    "description": "获取最近的事件列表。可按状态过滤，返回事件摘要。",
    "parameters": {
      "type": "object",
      "properties": {
        "state": {
          "type": "string",
          "enum": ["warning", "incident", "recovery", "stable"],
          "description": "按状态过滤，不填则返回所有状态"
        },
        "limit": {
          "type": "integer",
          "description": "返回数量，默认 10"
        }
      }
    }
  }
]`

// LoadToolDefinitions 加载 Tool 定义
// 从 toolsJSON 常量解析为 llm.ToolDefinition 列表
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

// ResetToolCache 重置 Tool 定义缓存（新增 Tool 后调用）
func ResetToolCache() {
	toolsCache = nil
}
