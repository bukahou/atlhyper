package control

// Command 与服务器端下发结构对齐
type Command struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Target map[string]string      `json:"target,omitempty"`
	Args   map[string]any         `json:"args,omitempty"`
	Idem   string                 `json:"idem,omitempty"`
}

type CommandSet struct {
	ClusterID string    `json:"clusterID"`
	RV        uint64    `json:"rv"`
	Commands  []Command `json:"commands"`
}

// AckResult 与 /ack 请求体里的 results[] 对齐
type AckResult struct {
    CommandID  string `json:"commandID"`
    Status     string `json:"status"`
    Message    string `json:"message,omitempty"`
    ErrorCode  string `json:"errorCode,omitempty"`
    StartedAt  string `json:"startedAt,omitempty"`
    FinishedAt string `json:"finishedAt,omitempty"`
    Attempt    int    `json:"attempt,omitempty"`
}