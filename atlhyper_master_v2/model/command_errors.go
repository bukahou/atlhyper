package model

import "errors"

// 创建与执行指令的错误类别。Gateway 按 errors.Is 映射成状态码，⛔ 不比对错误文案。
var (
	// ErrInvalidCommand 指令参数不合法（缺字段、目标名不合法、来源 × 动作不在名单里）
	ErrInvalidCommand = errors.New("指令参数不合法")
	// ErrClusterOffline 集群没有在线的 Agent：发了也没人取
	ErrClusterOffline = errors.New("集群没有在线的 Agent")
	// ErrEnqueueFailed 指令没能放进队列
	ErrEnqueueFailed = errors.New("指令未能入队")
)
