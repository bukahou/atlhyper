package gateway

import "errors"

// ErrMasterRejectedIdentity Master 以 401 / 403 拒绝了请求：
// 令牌缺失或无效（401），或令牌所属的集群与本 Agent 不符（403）。
//
// 与网络故障区分开：这类失败重试不会自己恢复，需要检查 AGENT_MASTER_TOKEN 与 Master 上的集群登记。
var ErrMasterRejectedIdentity = errors.New("Master 拒绝了本 Agent 的身份（请检查 AGENT_MASTER_TOKEN 与 Master 上的集群登记）")
