// Package contract 指令总线各实现共用的约定：话题名单与错误类别（叶子包，避免 mq ↔ 实现循环引用）
package contract

import (
	"context"
	"errors"
	"fmt"
)

// 话题：每个集群每个话题一条队列，互不阻塞
const (
	TopicOps = "ops" // Web 页面发起的操作与查询、部署器
	TopicAI  = "ai"  // AI 工具调用
)

var (
	// ErrUnknownTopic 话题不在名单里（不为它建队列）
	ErrUnknownTopic = errors.New("未知的指令话题")
	// ErrQueueFull 队列已满：没人取的指令不能无限堆积
	ErrQueueFull = errors.New("指令队列已满")
	// ErrWaitTimeout 等指令结果等到了上限（同时是 context.DeadlineExceeded）。
	// 「超时」是一种失败，⛔ 不用 (nil, nil) 表示
	ErrWaitTimeout = fmt.Errorf("等待指令结果超时: %w", context.DeadlineExceeded)
)

// CheckTopic 话题是否在名单里
func CheckTopic(topic string) error {
	switch topic {
	case TopicOps, TopicAI:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrUnknownTopic, topic)
	}
}
