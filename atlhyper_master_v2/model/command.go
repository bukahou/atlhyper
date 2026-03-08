// atlhyper_master_v2/model/command.go
// 指令相关请求/响应模型
package model

// CreateCommandRequest 创建指令请求
type CreateCommandRequest struct {
	ClusterID       string                 `json:"clusterId"`
	Action          string                 `json:"action"` // scale / restart / delete_pod / exec ...
	TargetKind      string                 `json:"targetKind,omitempty"`
	TargetNamespace string                 `json:"targetNamespace,omitempty"`
	TargetName      string                 `json:"targetName,omitempty"`
	Params          map[string]interface{} `json:"params,omitempty"`
	Source          string                 `json:"source,omitempty"` // web / ai
}

// CreateCommandResponse 创建指令响应
type CreateCommandResponse struct {
	CommandID string `json:"commandId"`
	Status    string `json:"status"`
}
