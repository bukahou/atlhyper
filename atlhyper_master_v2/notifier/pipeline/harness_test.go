package pipeline

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
	"AtlHyper/atlhyper_master_v2/notifier/template"
)

// 占位：不是任何真实系统的地址
const testWebhookURL = "https://hooks.example.com/services/T000/B000/placeholder-not-real"

var testStart = time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

// fakeClock 可拨动的时钟：测试里时间只在 advance 时前进
type fakeClock struct {
	mu      sync.Mutex
	current time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = c.current.Add(d)
}

// sentMessage 假发送器收到的一条消息
type sentMessage struct {
	receiverID int64
	subject    string
	body       string
}

// fakeSender 记录每次发送；可以让某个接收人的发送失败
type fakeSender struct {
	mu       sync.Mutex
	sent     []sentMessage
	calls    int             // 发送被调用的次数（含失败的）
	failWith map[int64]error // 接收人 id → 发送时返回的错误
	notify   chan struct{}   // 非 nil 时每次发送后写入一个信号
}

func (s *fakeSender) Send(_ context.Context, receiver *database.NotifyReceiver, msg *channel.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.notify != nil {
		defer func() { s.notify <- struct{}{} }()
	}
	if err := s.failWith[receiver.ID]; err != nil {
		return err
	}
	s.sent = append(s.sent, sentMessage{receiverID: receiver.ID, subject: msg.Subject, body: msg.Body})
	return nil
}

func (s *fakeSender) messages() []sentMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]sentMessage(nil), s.sent...)
}

func (s *fakeSender) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func (s *fakeSender) fail(receiverID int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith == nil {
		s.failWith = map[int64]error{}
	}
	if err == nil {
		delete(s.failWith, receiverID)
		return
	}
	s.failWith[receiverID] = err
}

// harness 一条接在真实 SQLite（临时文件）上的管道，时钟与发送器是假的
type harness struct {
	t      *testing.T
	db     *database.DB
	clock  *fakeClock
	sender *fakeSender
	p      *AlertPipeline
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	if err := repo.SetEncryptionSecret("test-secret-not-real"); err != nil {
		t.Fatal(err)
	}
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "t.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())

	h := &harness{t: t, db: db, clock: &fakeClock{current: testStart}, sender: &fakeSender{}}
	h.p = h.newPipeline()
	return h
}

func (h *harness) newPipeline() *AlertPipeline {
	renderer, err := template.NewAlertRenderer("", nil)
	if err != nil {
		h.t.Fatal(err)
	}
	return NewAlertPipeline(Config{
		Repos: Repos{
			Policy: h.db.NotifyPolicy, Alerts: h.db.NotifyAlert, Deliveries: h.db.NotifyDelivery,
			Receivers: h.db.NotifyReceiver, Rules: h.db.NotifyRule, Silences: h.db.NotifySilence,
		},
		Sender:   h.sender,
		Renderer: renderer,
		Now:      h.clock.now,
		// 测试里串行发送：假发送器按调用顺序记录消息，顺序要稳定
		SendConcurrency: 1,
	})
}

// restart 模拟 Master 重启：丢掉内存状态，在同一个库上新建管道并恢复
func (h *harness) restart() {
	h.t.Helper()
	h.p = h.newPipeline()
	h.p.restoreFromStore()
}

// submit 同步处理一条告警（绕过队列，测试里没有并发）
func (h *harness) submit(a alert.Alert) {
	h.t.Helper()
	h.p.process(submission{alert: &a})
}

// advance 时钟前进并跑一轮
func (h *harness) advance(d time.Duration) {
	h.t.Helper()
	h.clock.advance(d)
	h.p.tick()
}

func (h *harness) addReceiver(name string) int64 {
	h.t.Helper()
	receiver := &database.NotifyReceiver{Name: name, Type: database.NotifyReceiverSlack, Enabled: true,
		Slack: &database.SlackConfig{WebhookURL: testWebhookURL}}
	if err := h.db.NotifyReceiver.Create(context.Background(), receiver); err != nil {
		h.t.Fatal(err)
	}
	return receiver.ID
}

func (h *harness) setPolicy(change func(*database.NotifyPolicy)) {
	h.t.Helper()
	ctx := context.Background()
	policy, err := h.db.NotifyPolicy.Get(ctx)
	if err != nil {
		h.t.Fatal(err)
	}
	change(policy)
	if err := h.db.NotifyPolicy.Save(ctx, policy); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) addRule(rule *database.NotifyRule) {
	h.t.Helper()
	if err := h.db.NotifyRule.Create(context.Background(), rule); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) addSilence(silence *database.NotifySilence) int64 {
	h.t.Helper()
	silence.Comment, silence.CreatedBy = "维护", "admin"
	if err := h.db.NotifySilence.Create(context.Background(), silence); err != nil {
		h.t.Fatal(err)
	}
	return silence.ID
}

func (h *harness) storedAlert(fingerprint string) *database.NotifyAlert {
	h.t.Helper()
	row, err := h.db.NotifyAlert.Get(context.Background(), fingerprint)
	if err != nil {
		h.t.Fatal(err)
	}
	if row == nil {
		h.t.Fatalf("库里没有告警 %s", fingerprint)
	}
	return row
}

// deliveries 某条告警的送达记录，按写入顺序
func (h *harness) deliveries(fingerprint string) []*database.NotifyDelivery {
	h.t.Helper()
	rows, err := h.db.NotifyDelivery.List(context.Background(), database.NotifyDeliveryQueryOpts{Fingerprint: fingerprint})
	if err != nil {
		h.t.Fatal(err)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// deliveryKinds 某条告警各条送达记录的「种类/状态」
func (h *harness) deliveryKinds(fingerprint string) []string {
	h.t.Helper()
	var kinds []string
	for _, row := range h.deliveries(fingerprint) {
		kinds = append(kinds, row.Kind+"/"+row.Status)
	}
	return kinds
}

func (h *harness) wantDeliveries(fingerprint string, want ...string) {
	h.t.Helper()
	got := h.deliveryKinds(fingerprint)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		h.t.Errorf("%s 的送达记录是 %v，期望 %v", fingerprint, got, want)
	}
}

func (h *harness) wantSent(count int) []sentMessage {
	h.t.Helper()
	messages := h.sender.messages()
	if len(messages) != count {
		h.t.Fatalf("应发出 %d 条消息，实际 %d 条：%+v", count, len(messages), messages)
	}
	return messages
}

// ==================== 告警构造 ====================

func incidentAlert(id, clusterID string, severity alert.Severity) alert.Alert {
	subject := "shop/deployment/" + id
	return alert.Alert{
		Fingerprint: alert.IncidentFingerprint(id), Source: alert.SourceIncident, ClusterID: clusterID,
		Severity: severity, Status: alert.StatusFiring, Kind: alert.KindFiring,
		Title: subject + " 风险异常", Summary: "风险持续偏高，已升级为事件",
		Labels:  map[string]string{alert.LabelNamespace: "shop", alert.LabelSubject: subject, alert.LabelSubjectType: "deployment"},
		Details: map[string]string{alert.DetailIncidentID: id},
	}
}

func withKind(a alert.Alert, kind alert.Kind) alert.Alert {
	a.Kind = kind
	return a
}

func resolved(a alert.Alert) alert.Alert {
	a.Status, a.Kind, a.Summary = alert.StatusResolved, alert.KindResolved, "风险已回落"
	return a
}

func heartbeatAlert(clusterID string) alert.Alert {
	return alert.Alert{
		Fingerprint: alert.HeartbeatFingerprint(clusterID), Source: alert.SourceHeartbeat, ClusterID: clusterID,
		Severity: alert.SeverityCritical, Status: alert.StatusFiring, Kind: alert.KindFiring,
		Title: "Agent 离线", Summary: "Agent 心跳超时",
	}
}

var errSendFailed = errors.New("对端返回 500")
