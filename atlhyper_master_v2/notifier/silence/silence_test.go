package silence

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

var t0 = time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

func window(id int64, clusterID, namespace, subject string) *database.NotifySilence {
	return &database.NotifySilence{ID: id, ClusterID: clusterID, Namespace: namespace, Subject: subject,
		StartsAt: t0, EndsAt: t0.Add(2 * time.Hour)}
}

func incidentAlert(clusterID, namespace, subject string) *alert.Alert {
	return &alert.Alert{Source: alert.SourceIncident, ClusterID: clusterID,
		Labels: map[string]string{alert.LabelNamespace: namespace, alert.LabelSubject: subject}}
}

func TestState(t *testing.T) {
	silence := window(1, "", "", "")
	cases := []struct {
		at   time.Time
		want string
	}{
		{t0.Add(-time.Second), StatePending},
		{t0, StateActive},
		{t0.Add(time.Hour), StateActive},
		{t0.Add(2 * time.Hour), StateExpired}, // 结束时刻本身已不在静默内
		{t0.Add(3 * time.Hour), StateExpired},
	}
	for _, c := range cases {
		if got := State(silence, c.at); got != c.want {
			t.Errorf("State(%v) = %q，期望 %q", c.at.Sub(t0), got, c.want)
		}
	}
}

func TestMatch(t *testing.T) {
	during := t0.Add(time.Hour)
	api := incidentAlert("prod", "shop", "shop/deployment/api")
	heartbeat := &alert.Alert{Source: alert.SourceHeartbeat, ClusterID: "prod"}

	cases := []struct {
		name    string
		silence *database.NotifySilence
		alert   *alert.Alert
		at      time.Time
		want    int64
	}{
		{"全空条件匹配一切", window(1, "", "", ""), api, during, 1},
		{"集群一致", window(2, "prod", "", ""), api, during, 2},
		{"集群不一致", window(3, "test", "", ""), api, during, 0},
		{"集群 + 命名空间", window(4, "prod", "shop", ""), api, during, 4},
		{"命名空间不一致", window(5, "prod", "pay", ""), api, during, 0},
		{"主体精确匹配", window(6, "", "", "shop/deployment/api"), api, during, 6},
		{"主体不是前缀匹配", window(7, "", "", "shop/deployment/ap"), api, during, 0},
		{"未开始的静默不生效", window(8, "prod", "", ""), api, t0.Add(-time.Minute), 0},
		{"已过期的静默不生效", window(9, "prod", "", ""), api, t0.Add(2 * time.Hour), 0},
		{"心跳告警：只限定集群的静默生效", window(10, "prod", "", ""), heartbeat, during, 10},
		{"心跳告警：限定了命名空间的静默不生效", window(11, "prod", "shop", ""), heartbeat, during, 0},
		{"心跳告警：限定了主体的静默不生效", window(12, "prod", "", "shop/deployment/api"), heartbeat, during, 0},
		{"心跳告警：别的集群的静默不生效", window(13, "test", "", ""), heartbeat, during, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Match([]*database.NotifySilence{c.silence}, c.alert, c.at); got != c.want {
				t.Errorf("Match = %d，期望 %d", got, c.want)
			}
		})
	}
}

// 多条都命中时取 id 最小的那条：结果稳定，不随列表顺序变化
func TestMatch_PicksLowestIDAmongMatches(t *testing.T) {
	api := incidentAlert("prod", "shop", "shop/deployment/api")
	silences := []*database.NotifySilence{window(9, "prod", "", ""), window(4, "", "", ""), window(7, "test", "", "")}
	if got := Match(silences, api, t0.Add(time.Minute)); got != 4 {
		t.Errorf("应取 id 最小的命中项 4，实际 %d", got)
	}
	if got := Match(nil, api, t0); got != 0 {
		t.Errorf("没有静默时应返回 0，实际 %d", got)
	}
}

// 还没开始就被提前结束的静默（建错了撤销）：结束时间早于开始时间，任何时刻都是「已结束」，不再生效
func TestState_ExpiredBeforeStart(t *testing.T) {
	cancelled := &database.NotifySilence{ID: 1, StartsAt: t0.Add(time.Hour), EndsAt: t0}
	for _, at := range []time.Time{t0, t0.Add(30 * time.Minute), t0.Add(time.Hour), t0.Add(2 * time.Hour)} {
		if got := State(cancelled, at); got != StateExpired {
			t.Errorf("State(%v) = %q，期望 expired", at.Sub(t0), got)
		}
		if Matches(cancelled, incidentAlert("prod", "shop", "shop/deployment/api"), at) {
			t.Errorf("被撤销的静默不应生效（%v）", at.Sub(t0))
		}
	}
}
