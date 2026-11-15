// atlhyper_master_v2/notifier/silence/silence.go
// 静默匹配（纯函数）：给定时刻，一条告警是否落在某条静默里
package silence

import (
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// 静默的状态，由时间算出，不落库
const (
	StatePending = "pending" // 还没开始
	StateActive  = "active"  // 生效中
	StateExpired = "expired" // 已结束
)

// State 静默在给定时刻的状态。区间是 [StartsAt, EndsAt)：结束时刻本身已不在静默内。
//
// 先看结束时间：还没开始就被提前结束的静默（EndsAt 早于 StartsAt）是「已结束」，不会再生效
func State(silence *database.NotifySilence, now time.Time) string {
	switch {
	case !now.Before(silence.EndsAt):
		return StateExpired
	case now.Before(silence.StartsAt):
		return StatePending
	default:
		return StateActive
	}
}

// Matches 这条静默此刻是否压住这条告警：静默生效中，且集群、命名空间、主体三个条件都满足。
//
// 空条件视为满足；填了的条件与告警的对应项逐字比较（主体是精确匹配，不是前缀）。
// 心跳告警没有命名空间与主体，所以只有不限定这两项的静默（只按集群）才会压住它 ——
// 维护某个命名空间不应把「整个集群失联」一并压掉
func Matches(silence *database.NotifySilence, a *alert.Alert, now time.Time) bool {
	if State(silence, now) != StateActive {
		return false
	}
	if silence.ClusterID != "" && silence.ClusterID != a.ClusterID {
		return false
	}
	if silence.Namespace != "" && silence.Namespace != a.Label(alert.LabelNamespace) {
		return false
	}
	if silence.Subject != "" && silence.Subject != a.Label(alert.LabelSubject) {
		return false
	}
	return true
}

// Match 返回压住这条告警的静默的 id；没有则为 0。
// 多条都命中时取 id 最小的，结果不随列表顺序变化
func Match(silences []*database.NotifySilence, a *alert.Alert, now time.Time) int64 {
	var matched int64
	for _, silence := range silences {
		if !Matches(silence, a, now) {
			continue
		}
		if matched == 0 || silence.ID < matched {
			matched = silence.ID
		}
	}
	return matched
}
