// Package mcpserver MCP 入口：把 AI 工具注册表里对外开放的工具，经 MCP（Streamable HTTP，无状态）
// 提供给外部的 AI 客户端。
//
// 入口有自己的监听端口，与 Web API（gateway）、Agent 入口（agentsdk）并列。
// 每个请求都要带访问令牌；工具清单按调用者当前的角色等级过滤，调用时再查一遍权限；每次工具调用记审计。
package mcpserver

import (
	"context"
	"errors"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/agent"
)

// Caller 经访问令牌识别出的调用者
type Caller struct {
	UserID    int64
	Username  string
	Level     int // 账号当前的角色等级
	TokenID   int64
	TokenName string
	ExpiresAt time.Time // 零值 = 不过期
}

// ErrInvalidToken 令牌无效（不存在、已吊销、已过期、账号停用）
var ErrInvalidToken = errors.New("访问令牌无效")

// TokenAuthenticator 校验访问令牌。
// 令牌无效时返回能用 errors.Is 认出 ErrInvalidToken 的错误；其他错误表示暂时无法校验（回 5xx，不当成无效）
type TokenAuthenticator interface {
	Authenticate(ctx context.Context, plaintext string) (Caller, error)
}

// ClusterLister 正在上报的集群
type ClusterLister interface {
	ListClusters(ctx context.Context) ([]agent.ClusterInfo, error)
}

// AuditRecorder 写一条审计
type AuditRecorder interface {
	Create(ctx context.Context, entry *database.AuditLog) error
}

// Config MCP 入口的配置与依赖
type Config struct {
	Port    int
	Version string // 报给客户端的服务端版本

	Tools         *toolkit.Registry
	Authenticator TokenAuthenticator
	Clusters      ClusterLister
	Audit         AuditRecorder
	// Allowed 某一级是否拥有某项权限（网关的权限表）
	Allowed func(level int, permission string) bool
	// Levels 全部角色等级：启动时为每一级备好它看得到的工具清单
	Levels []int

	RateLimitPerMinute int           // 每枚令牌每分钟的请求数；0 用默认值
	MaxResultRunes     int           // 单次工具结果的字符上限；0 用默认值
	MaxBodyBytes       int64         // 请求体上限；0 用默认值
	CallTimeout        time.Duration // 单次工具调用的时限；0 用默认值
}
