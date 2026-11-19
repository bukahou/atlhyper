package notifier

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
	"AtlHyper/atlhyper_master_v2/notifier/pipeline"
	agentmodel "AtlHyper/model_v3/agent"
)

// recordingSender 记录发出的消息，每发一条往 delivered 里写一个信号
type recordingSender struct {
	mu        sync.Mutex
	messages  []*channel.Message
	delivered chan struct{}
	err       error
}

func (s *recordingSender) Send(_ context.Context, _ *database.NotifyReceiver, msg *channel.Message) error {
	s.mu.Lock()
	s.messages = append(s.messages, msg)
	s.mu.Unlock()
	if s.delivered != nil {
		s.delivered <- struct{}{}
	}
	return s.err
}

type noAgents struct{}

func (noAgents) ListAgents() ([]agentmodel.AgentInfo, error) { return nil, nil }

func newTestManager(t *testing.T, publicURL string) (*Manager, *database.DB, *recordingSender) {
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

	sender := &recordingSender{delivered: make(chan struct{}, 8)}
	manager, err := NewAlertManager(Config{
		Repos: pipeline.Repos{
			Policy: db.NotifyPolicy, Alerts: db.NotifyAlert, Deliveries: db.NotifyDelivery,
			Receivers: db.NotifyReceiver, Rules: db.NotifyRule, Silences: db.NotifySilence,
		},
		Events:               db.Event,
		Agents:               noAgents{},
		PublicURL:            publicURL,
		Sender:               sender,
		PipelineTickInterval: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager, db, sender
}

func addTestReceiver(t *testing.T, db *database.DB, webhookURL string) int64 {
	t.Helper()
	receiver := &database.NotifyReceiver{Name: "值班频道", Type: database.NotifyReceiverSlack, Enabled: true,
		Slack: &database.SlackConfig{WebhookURL: webhookURL}}
	if err := db.NotifyReceiver.Create(context.Background(), receiver); err != nil {
		t.Fatal(err)
	}
	return receiver.ID
}

// 从事件变化到消息发出的整条链路：订阅回调只入队 → 管道协程按策略转成告警 → 分组 → 发送 → 送达记录
func TestIncidentTransitionReachesReceiver(t *testing.T) {
	manager, db, sender := newTestManager(t, "https://atlhyper.example.com")
	ctx := context.Background()
	addTestReceiver(t, db, "https://hooks.example.com/services/T000/B000/placeholder-not-real")
	policy := database.DefaultNotifyPolicy()
	policy.GroupWaitSeconds = 0 // 不等分组：下一轮就发
	if err := db.NotifyPolicy.Save(ctx, policy); err != nil {
		t.Fatal(err)
	}

	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()

	manager.OnIncidentTransition(aiops.IncidentTransition{
		Kind: aiops.TransitionEscalated, IncidentID: "inc-123", ClusterID: "prod", SubjectKey: "shop/deployment/api",
		RootCause: "shop/pod/api-7d9f", Members: []string{"shop/pod/api-7d9f"},
		FromState: aiops.StateWarning, ToState: aiops.StateIncident, Severity: "high", PeakRisk: 0.72,
		Occurrences24h: 1, ReachedIncident: true, At: time.Now(),
	})

	select {
	case <-sender.delivered:
	case <-time.After(10 * time.Second):
		t.Fatal("事件升级后没有发出通知")
	}
	sender.mu.Lock()
	message := sender.messages[0]
	sender.mu.Unlock()
	if !strings.Contains(message.Subject, "[prod] 高 · shop/deployment/api") || !strings.Contains(message.Body, "触发") ||
		!strings.Contains(message.Body, "https://atlhyper.example.com/aiops/incidents?incident=inc-123") {
		t.Errorf("通知内容不对:\n%s\n%s", message.Subject, message.Body)
	}

	stored, err := db.NotifyAlert.Get(ctx, "incident:inc-123")
	if err != nil || stored == nil || stored.Status != "firing" || stored.Severity != "high" {
		t.Errorf("告警应已落库: %+v (%v)", stored, err)
	}
}

func TestTestReceiver(t *testing.T) {
	manager, db, sender := newTestManager(t, "")
	ctx := context.Background()
	receiverID := addTestReceiver(t, db, "https://hooks.example.com/services/T000/B000/placeholder-not-real")

	if err := manager.TestReceiver(ctx, receiverID); err != nil {
		t.Fatalf("测试发送应成功: %v", err)
	}
	if len(sender.messages) != 1 || !strings.Contains(sender.messages[0].Subject, "测试") || !strings.Contains(sender.messages[0].Body, "值班频道") {
		t.Errorf("测试消息不对: %+v", sender.messages)
	}

	if err := manager.TestReceiver(ctx, 999); !errors.Is(err, ErrReceiverNotFound) {
		t.Errorf("不存在的接收人应返回 ErrReceiverNotFound，实际 %v", err)
	}

	sender.err = errors.New("对端返回 500")
	if err := manager.TestReceiver(ctx, receiverID); err == nil || !strings.Contains(err.Error(), "对端返回 500") {
		t.Errorf("发送失败应带出原因，实际 %v", err)
	}
}

// 测试发送失败的原因会原样返回给界面：里面不能有 Webhook 地址（用真实的发送器、连不上的本机地址）
func TestTestReceiver_FailureDoesNotLeakWebhookURL(t *testing.T) {
	manager, db, _ := newTestManager(t, "")
	manager.sender = channel.NewReceiverSender(db.NotifySMTP)

	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	webhookURL := closed.URL + "/services/T000/B000/placeholder-not-real"
	closed.Close()
	receiverID := addTestReceiver(t, db, webhookURL)

	err := manager.TestReceiver(context.Background(), receiverID)
	if err == nil {
		t.Fatal("连不上的地址应当失败")
	}
	for _, leaked := range []string{webhookURL, "placeholder-not-real", "/services/T000", strings.TrimPrefix(closed.URL, "http://")} {
		if strings.Contains(err.Error(), leaked) {
			t.Errorf("错误信息泄漏了 %q：%s", leaked, err)
		}
	}
}

// 配置解不开或不完整的接收人：报配置不完整，不去发送
func TestTestReceiver_IncompleteConfig(t *testing.T) {
	manager, db, _ := newTestManager(t, "")
	manager.sender = channel.NewReceiverSender(db.NotifySMTP)
	receiverID := addTestReceiver(t, db, "")

	if err := manager.TestReceiver(context.Background(), receiverID); !errors.Is(err, ErrReceiverConfigIncomplete) {
		t.Errorf("没有 Webhook 地址应返回 ErrReceiverConfigIncomplete，实际 %v", err)
	}
}

func TestSanitizePublicURL(t *testing.T) {
	cases := map[string]string{
		"":                                   "",
		"https://atlhyper.example.com":       "https://atlhyper.example.com",
		"https://atlhyper.example.com/":      "https://atlhyper.example.com",
		"http://atlhyper.example.com:8080/x": "http://atlhyper.example.com:8080/x",
		"atlhyper.example.com":               "", // 没有协议：不是能点开的链接
		"javascript:alert(1)":                "",
		"ftp://atlhyper.example.com":         "",
		"https://":                           "",
	}
	for input, want := range cases {
		if got := sanitizePublicURL(input); got != want {
			t.Errorf("sanitizePublicURL(%q) = %q，期望 %q", input, got, want)
		}
	}
}

func TestStartStopIsIdempotent(t *testing.T) {
	manager, _, _ := newTestManager(t, "")
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	manager.Stop()
	manager.Stop()
}
