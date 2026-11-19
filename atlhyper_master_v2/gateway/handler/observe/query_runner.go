package observe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/singleflight"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/command"
)

// queryTimeout 经指令通道查询时等待 Agent 结果的上限
const queryTimeout = 30 * time.Second

// errQueryTimeout 等不到 Agent 的结果
var errQueryTimeout = fmt.Errorf("等待 Agent 查询结果超时: %w", context.DeadlineExceeded)

// agentQueryError Agent 执行了查询但失败（带 Agent 给的原因）
type agentQueryError struct{ reason string }

func (e *agentQueryError) Error() string { return e.reason }

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
func (q *commandQuery) run(ctx context.Context, clusterID, action string, params map[string]any, ttl time.Duration) (json.RawMessage, error) {
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
		if result == nil {
			return nil, errQueryTimeout
		}
		if !result.Success {
			reason := result.Error
			if reason == "" {
				reason = "查询失败"
			}
			return nil, &agentQueryError{reason: reason}
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
	var agentErr *agentQueryError
	if errors.As(err, &agentErr) {
		handler.WriteError(w, http.StatusBadGateway, agentErr.reason)
		return
	}
	handler.WriteCommandError(w, err)
}
