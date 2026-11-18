// atlhyper_master_v2/aiops/enricher/background_failure.go
// 后台分析失败时的日志级别：AI 没配是配置状态，不是故障
package enricher

import (
	"errors"
	"sync/atomic"

	"AtlHyper/atlhyper_master_v2/ai"
)

const (
	failureLevelError    = "error"     // 真的失败了：照常报错
	failureLevelWarnOnce = "warn_once" // AI 没配：提示一次
	failureLevelSilent   = "silent"    // AI 没配，已经提示过
)

// unconfiguredNotice 记下「AI 没配」是否已经提示过。成功一次后清零，之后再没配会再提示一次
type unconfiguredNotice struct {
	warned atomic.Bool
}

// levelFor 这次失败该用什么级别记日志
func (n *unconfiguredNotice) levelFor(err error) string {
	if !errors.Is(err, ai.ErrRoleNotAssigned) {
		return failureLevelError
	}
	if n.warned.Swap(true) {
		return failureLevelSilent
	}
	return failureLevelWarnOnce
}

// succeeded 后台分析成功了一次
func (n *unconfiguredNotice) succeeded() {
	n.warned.Store(false)
}
