// Package gateway 封装与 Master 的通信
//
// Gateway 层负责 Agent 与 Master 之间的所有 HTTP 通信。
// 上层只依赖 MasterGateway 接口，不感知具体的 HTTP 实现细节。
//
// 通信模式:
//   - Agent 主动发起所有请求 (推送快照、拉取指令、上报结果)
//   - Master 被动响应
//   - 长轮询获取指令 (减少轮询频率)
//
// 架构位置:
//
//	Scheduler
//	    ↓
//	Service
//	    ↓
//	Gateway (本包) ← HTTP 通信
//	    ↓
//	Master 服务
package gateway

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/model_v2"
)

// MasterGateway Master 通信接口
//
// 封装所有与 Master 的通信，提供以下功能:
//   - 推送集群快照
//   - 拉取待执行指令
//   - 上报执行结果
//   - 心跳保活
type MasterGateway interface {
	// PushSnapshot 推送集群快照到 Master
	//
	// Agent 定时调用此方法，将采集的快照推送给 Master。
	// 数据使用 JSON + Gzip 压缩传输。
	//
	// HTTP: POST /agent/snapshot
	// Header: X-Cluster-ID, Content-Encoding: gzip
	PushSnapshot(ctx context.Context, snapshot *model_v2.ClusterSnapshot) error

	// PollCommands 从 Master 拉取待执行指令
	//
	// 使用长轮询方式，Master 会 hold 请求直到有指令或超时。
	// 204 No Content 表示无指令。
	// topic: "ops" 或 "ai"，分别对应系统操作和 AI 查询队列。
	//
	// HTTP: GET /agent/commands?cluster_id=xxx&topic=yyy
	PollCommands(ctx context.Context, topic string) ([]model_v2.Command, error)

	// ReportResult 上报指令执行结果到 Master
	//
	// 每执行完一个指令，立即上报结果。
	//
	// HTTP: POST /agent/result
	ReportResult(ctx context.Context, result *model.Result) error

	// Heartbeat 心跳
	//
	// 定时发送心跳，让 Master 知道 Agent 存活。
	//
	// HTTP: POST /agent/heartbeat
	Heartbeat(ctx context.Context) error

	// PushSLOMetrics 推送 SLO 指标到 Master
	//
	// Agent 定时调用此方法，将采集的 Ingress 指标推送给 Master。
	// 同时推送 IngressRoute 映射信息，用于 domain/path 维度的展示。
	//
	// HTTP: POST /agent/slo
	PushSLOMetrics(ctx context.Context, metrics *model.IngressMetrics, routes []model.IngressRouteInfo) error
}
