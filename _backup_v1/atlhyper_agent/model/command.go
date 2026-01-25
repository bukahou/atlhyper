// atlhyper_agent/model/command.go
// Agent 内部命令模型（Master 下发 + 本地执行 + 回执）
package model

// Command 是 Master 下发给 Agent 的命令结构
type Command struct {
	ID     string            `json:"id"`
	Type   string            `json:"type"`             // PodRestart / NodeCordon / NodeUncordon / UpdateImage / ScaleWorkload / PodGetLogs
	Target map[string]string `json:"target,omitempty"` // 资源定位：ns/pod 或 ns/kind/name 或 node
	Args   map[string]any    `json:"args,omitempty"`   // 参数：newImage/replicas 等
	Idem   string            `json:"idem,omitempty"`   // 幂等键（用于 Ack/去重）
}

// CommandSet 是 Master 批量下发的命令集合
type CommandSet struct {
	ClusterID string    `json:"clusterID"`
	RV        uint64    `json:"rv"`
	Commands  []Command `json:"commands"`
}

// Result 是命令执行结果（用于本地返回）
type Result struct {
	CommandID string `json:"commandID"`
	Idem      string `json:"idem,omitempty"`
	Status    string `json:"status"`    // Succeeded / Failed / Skipped
	ErrorCode string `json:"errorCode"` // Forbidden / NotFound / Conflict...
	Message   string `json:"message"`
}

// AckResult 是回执给 Master 的执行结果
type AckResult struct {
	CommandID  string `json:"commandID"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	ErrorCode  string `json:"errorCode,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Attempt    int    `json:"attempt,omitempty"`
}
