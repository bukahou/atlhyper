package observe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/sync/singleflight"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/command"
)

var log = logger.Module("Observe")

// queryTimeout 经指令通道查询时等待 Agent 结果的上限
const queryTimeout = 30 * time.Second

// agentQueryError Agent 执行了查询但失败。只带失败类别：原始错误在查询时已写进日志，
// ⛔ 不随错误往外传（这些接口访客也能调，原文里可能有 SQL 片段与内部地址）
type agentQueryError struct{ code string }

func (e *agentQueryError) Error() string { return "agent query failed: " + e.code }

// commandQuery 经指令通道做只读查询。
// 缓存期内相同的查询直接用缓存；同时到达的相同查询只发一条指令、共享结果 ——
// 这些接口访客也能调，不合并就是一个请求一条指令。
type commandQuery struct {
	ops   ObserveOps
	cache *observeCache
	group singleflight.Group
}

func newCommandQuery(ops ObserveOps) *commandQuery {
	return &commandQuery{ops: ops, cache: newObserveCache()}
}

// run 执行查询，返回 Agent 给的 JSON。ttl 为 0 时不缓存；失败结果一律不缓存。
// 指令的等待不随某一个请求的断开而取消（结果是几个请求共享的），上限是 queryTimeout。
func (q *commandQuery) run(ctx context.Context, clusterID, action string, query command.QueryParams, ttl time.Duration) (json.RawMessage, error) {
	// 发送前先校验：不合法的查询在这里就拒绝（400），不进指令队列
	if query == nil {
		return nil, &invalidQueryError{reason: "缺少查询参数"}
	}
	if err := query.Validate(); err != nil {
		return nil, &invalidQueryError{reason: err.Error()}
	}
	params := query.ToParams()
	key := buildCacheKey(clusterID, action, params)
	if data, ok := q.cache.get(key); ok {
		return data, nil
	}
	v, err, _ := q.group.Do(key, func() (any, error) {
		if data, ok := q.cache.get(key); ok {
			return data, nil
		}
		result, err := q.ops.ExecuteCommandSync(context.WithoutCancel(ctx), &model.CreateCommandRequest{
			ClusterID: clusterID,
			Action:    action,
			Params:    params,
			Source:    command.SourceWeb,
		}, queryTimeout)
		if err != nil {
			return nil, err
		}
		if !result.Success {
			code := result.Code
			if !command.IsKnownResultCode(code) {
				code = command.CodeFailed
			}
			log.Warn("Agent 查询失败", "cluster", clusterID, "action", action, "command", result.CommandID, "code", code, "err", result.Error)
			return nil, &agentQueryError{code: code}
		}
		data := json.RawMessage(result.Output)
		if ttl > 0 {
			q.cache.set(key, data, ttl)
		}
		return data, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(json.RawMessage), nil
}

// writeQueryError 查询错误 → 状态码：Agent 查询失败 502，其余按指令错误类别（超时 504 等）
func writeQueryError(w http.ResponseWriter, err error) {
	var invalid *invalidQueryError
	if errors.As(err, &invalid) {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": invalid.reason, "code": command.CodeInvalidParams})
		return
	}
	var agentErr *agentQueryError
	if errors.As(err, &agentErr) {
		status, message := agentFailureResponse(agentErr.code)
		handler.WriteJSON(w, status, map[string]string{"error": message, "code": agentErr.code})
		return
	}
	handler.WriteCommandError(w, err)
}

// agentFailureResponse Agent 查询失败的类别 → 状态码与通用说明（界面按 code 出中日文提示，这里的说明是兜底）
func agentFailureResponse(code string) (int, string) {
	switch code {
	case command.CodeNotConfigured:
		return http.StatusConflict, "该集群的 Agent 没有接入 ClickHouse，无法查询"
	case command.CodeInvalidParams:
		return http.StatusBadRequest, "查询条件不合法"
	case command.CodeTimeout:
		return http.StatusGatewayTimeout, "查询超时，请缩小时间范围后重试"
	case command.CodeUnavailable:
		return http.StatusServiceUnavailable, "暂时连不上数据源，请稍后重试"
	case command.CodeRejected:
		return http.StatusBadGateway, "Agent 拒绝执行这次查询"
	default:
		return http.StatusBadGateway, "查询失败"
	}
}

// invalidQueryError 查询参数不合法（由请求方修正）
type invalidQueryError struct{ reason string }

func (e *invalidQueryError) Error() string { return e.reason }
