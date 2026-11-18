// atlhyper_master_v2/notifier/pipeline/state.go
// 管道的内存状态：告警、分组、批次，以及一轮处理用到的配置快照
package pipeline

import (
	"sort"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/routing"
)

// alertState 一条告警及其通知状态
type alertState struct {
	alert alert.Alert

	// 落库的通知状态
	lastNotifiedAt       time.Time      // 最近一次进入发送（含「没有接收人」）的时刻
	lastNotifiedKind     alert.Kind     // 本次持续期间最近一次已通知的变化；空 = 这次持续还没通知过任何人
	lastNotifiedSeverity alert.Severity // 那次通知时的严重度
	notifyCount          int
	suppressedBy         string
	silencedBy           int64

	// 只在内存里
	lastAttemptAt time.Time  // 最近一次为它做通知决定的时刻；重复提醒从这里起算
	wantKind      alert.Kind // 等待路由的变化；空 = 没有
	force         bool       // 这次路由跳过去重（静默 / 抑制解除后按新告警发出）
	queued        *groupKey  // 它在哪个分组里有待发的内容；nil = 没有
}

// held 是否被静默或抑制压住
func (s *alertState) held() bool { return s.silencedBy != 0 || s.suppressedBy != "" }

// announced 这次持续是否已经通知过人
func (s *alertState) announced() bool { return s.lastNotifiedKind != "" }

// row 转成库里的一行
func (s *alertState) row() *database.NotifyAlert {
	row := &database.NotifyAlert{
		Fingerprint: s.alert.Fingerprint, Source: string(s.alert.Source), ClusterID: s.alert.ClusterID,
		Severity: string(s.alert.Severity), Status: string(s.alert.Status), Kind: string(s.alert.Kind),
		Title: s.alert.Title, Summary: s.alert.Summary, Labels: s.alert.Labels, Details: s.alert.Details,
		StartsAt: s.alert.StartsAt, EndsAt: s.alert.EndsAt, UpdatedAt: s.alert.UpdatedAt,
		LastNotifiedKind: string(s.lastNotifiedKind), LastNotifiedSeverity: string(s.lastNotifiedSeverity),
		NotifyCount: s.notifyCount, SuppressedBy: s.suppressedBy, SilencedBy: s.silencedBy,
	}
	if !s.lastNotifiedAt.IsZero() {
		notifiedAt := s.lastNotifiedAt
		row.LastNotifiedAt = &notifiedAt
	}
	return row
}

// alertFromRow 库里的一行 → 告警对象
func alertFromRow(row *database.NotifyAlert) alert.Alert {
	return alert.Alert{
		Fingerprint: row.Fingerprint, Source: alert.Source(row.Source), ClusterID: row.ClusterID,
		Severity: alert.Severity(row.Severity), Status: alert.Status(row.Status), Kind: alert.Kind(row.Kind),
		Title: row.Title, Summary: row.Summary, Labels: row.Labels, Details: row.Details,
		StartsAt: row.StartsAt, EndsAt: row.EndsAt, UpdatedAt: row.UpdatedAt,
	}
}

// stateFromRow 库里的一行 → 内存状态
func stateFromRow(row *database.NotifyAlert) *alertState {
	state := &alertState{
		alert:                alertFromRow(row),
		lastNotifiedKind:     alert.Kind(row.LastNotifiedKind),
		lastNotifiedSeverity: alert.Severity(row.LastNotifiedSeverity),
		notifyCount:          row.NotifyCount,
		suppressedBy:         row.SuppressedBy,
		silencedBy:           row.SilencedBy,
	}
	if row.LastNotifiedAt != nil {
		state.lastNotifiedAt = *row.LastNotifiedAt
	}
	return state
}

// groupKey 分组的键：同一条规则（0 = 默认策略）+ 同一个集群
type groupKey struct {
	ruleID    int64
	clusterID string
}

// groupState 一个分组
type groupState struct {
	pending        map[string]alert.Kind // 指纹 → 待发的变化
	order          []string              // 待发内容的入组顺序
	firstPendingAt time.Time             // 这一批待发内容里最早的入组时刻
	lastFlushAt    time.Time             // 上一次发出的时刻；零值 = 还没发过
	timing         routing.Timing
}

// flushAt 这一批待发内容什么时候发：入组后等满「分组等待」；这组刚发过的话，还要距上次发出满「分组间隔」
func (g *groupState) flushAt() time.Time {
	at := g.firstPendingAt.Add(g.timing.GroupWait)
	if !g.lastFlushAt.IsZero() {
		if next := g.lastFlushAt.Add(g.timing.GroupInterval); next.After(at) {
			at = next
		}
	}
	return at
}

// batchItem 批次里的一条告警及这次要说的变化
type batchItem struct {
	fingerprint string
	kind        alert.Kind
}

// batchState 一个批次：一条消息发给一个接收人
type batchState struct {
	id            string
	seq           int
	receiverID    int64
	items         []batchItem
	attempts      int
	nextAttemptAt time.Time
}

// settings 一轮处理用到的配置快照
type settings struct {
	policy    *database.NotifyPolicy
	rules     []*database.NotifyRule
	receivers []*database.NotifyReceiver
	silences  []*database.NotifySilence
}

// receiver 按 id 找接收人；已不存在时返回 nil
func (s *settings) receiver(id int64) *database.NotifyReceiver {
	for _, receiver := range s.receivers {
		if receiver.ID == id {
			return receiver
		}
	}
	return nil
}

// route 当前配置下这条告警的路由结果
func (s *settings) route(a *alert.Alert) routing.Decision {
	return routing.Route(s.policy, s.rules, s.receivers, routing.TargetOf(a))
}

// sourceEnabled 这个来源的告警此刻是否允许通知。事件与旧的 K8s 事件告警各有开关；心跳没有开关
func (s *settings) sourceEnabled(source alert.Source) bool {
	switch source {
	case alert.SourceIncident:
		return s.policy.IncidentAlertsEnabled
	case alert.SourceK8sEvent:
		return s.policy.K8sEventAlertsEnabled
	default:
		return true
	}
}

// loadSettings 读出当前的策略、规则、接收人与静默
func (p *AlertPipeline) loadSettings() (*settings, error) {
	ctx, cancel := p.storeContext()
	defer cancel()

	policy, err := p.repos.Policy.Get(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := p.repos.Rules.List(ctx)
	if err != nil {
		return nil, err
	}
	receivers, err := p.repos.Receivers.List(ctx)
	if err != nil {
		return nil, err
	}
	silences, err := p.repos.Silences.List(ctx)
	if err != nil {
		return nil, err
	}
	return &settings{policy: policy, rules: rules, receivers: receivers, silences: silences}, nil
}

// sortedFingerprints 全部告警的指纹，按字典序 —— map 的遍历顺序不固定，处理顺序要稳定
func (p *AlertPipeline) sortedFingerprints() []string {
	fingerprints := make([]string, 0, len(p.alerts))
	for fingerprint := range p.alerts {
		fingerprints = append(fingerprints, fingerprint)
	}
	sort.Strings(fingerprints)
	return fingerprints
}

// sortedGroupKeys 全部分组的键，先按集群、再按规则
func (p *AlertPipeline) sortedGroupKeys() []groupKey {
	keys := make([]groupKey, 0, len(p.groups))
	for key := range p.groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].clusterID != keys[j].clusterID {
			return keys[i].clusterID < keys[j].clusterID
		}
		return keys[i].ruleID < keys[j].ruleID
	})
	return keys
}

// persist 把告警状态写回库。失败只记日志：内存里的状态仍然有效，下次变化时会再写
func (p *AlertPipeline) persist(state *alertState) {
	ctx, cancel := p.storeContext()
	defer cancel()
	if err := p.repos.Alerts.Upsert(ctx, state.row()); err != nil {
		log.Error("告警落库失败", "fingerprint", state.alert.Fingerprint, "err", err)
	}
}
