package command

import "encoding/json"

// 指令参数的共享定义。
//
// Command.Params 在传输上是一个 map，Master 填、Agent 读。两端各写各的键名很容易对不上
// （一端 tail_lines、另一端 tailLines，参数就静默失效）。带参数的指令在这里定义结构体：
// Master 用 ToParams 生成，Agent 用 ParseParams 解析，键名只有这一处。

// GetLogsParams get_logs 指令的参数
type GetLogsParams struct {
	Container    string `json:"container,omitempty"`    // 容器名，空则由 Agent 选主容器
	TailLines    int64  `json:"tailLines,omitempty"`    // 返回最后 N 行
	SinceSeconds int64  `json:"sinceSeconds,omitempty"` // 只返回最近 N 秒
	Timestamps   bool   `json:"timestamps,omitempty"`   // 每行带时间戳
	Previous     bool   `json:"previous,omitempty"`     // 读上一个容器实例的日志
}

// ToParams 转成 Command.Params
func (p GetLogsParams) ToParams() map[string]any {
	return toParams(p)
}

// ParseParams 把 Command.Params 解析进参数结构体
func ParseParams(params map[string]any, into any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}

func toParams(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	params := map[string]any{}
	if err := json.Unmarshal(raw, &params); err != nil {
		return map[string]any{}
	}
	return params
}
