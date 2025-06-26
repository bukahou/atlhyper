package dockerHub

import (
	"NeuroController/interfaces"
	"encoding/json"
	"fmt"
)

// DockerHubWebhookPayload 表示 DockerHub webhook 的基本结构
// 示例 payload:
//
//	{
//	  "push_data": { "tag": "default-media-v1.0.1" },
//	  "repository": { "repo_name": "bukahou/zgmf-x10a" }
//	}
type DockerHubWebhookPayload struct {
	PushData struct {
		Tag string `json:"tag"`
	} `json:"push_data"`

	Repository struct {
		RepoName string `json:"repo_name"`
	} `json:"repository"`
}

// ParseDockerHubWebhook 解析 DockerHub 的 webhook 数据，返回 repo 和 tag
// ParseAndApplyDockerHubWebhook 解析 DockerHub Webhook 并执行 Deployment 更新逻辑
func ParseAndApplyDockerHubWebhook(payload []byte) error {
	var data struct {
		Repository struct {
			RepoName string `json:"repo_name"`
		} `json:"repository"`
		PushData struct {
			Tag string `json:"tag"`
		} `json:"push_data"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("JSON 解析失败: %w", err)
	}

	repo := data.Repository.RepoName
	tag := data.PushData.Tag

	if repo == "" || tag == "" {
		return fmt.Errorf("缺少 repo 或 tag 信息")
	}

	return interfaces.UpdateDeploymentByTag(repo, tag)
}
