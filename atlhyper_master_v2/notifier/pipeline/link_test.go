package pipeline

import (
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// link 模拟引擎发来的关联变化（root 为空 = 解除）
func (h *harness) link(child, root, subject string) {
	h.t.Helper()
	update := linkUpdate{child: alert.IncidentFingerprint(child), subject: subject}
	if root != "" {
		update.root = alert.IncidentFingerprint(root)
	}
	h.p.process(submission{link: &update})
}

// 拓扑归并：源头告警持续时，归到它下面的告警被抑制，只发源头；源头的消息带影响范围
func TestRootIncidentSuppressesLinked(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.link("svc", "deploy", "shop/service/api")
	h.link("db-svc", "deploy", "shop/service/db")
	h.submit(incidentAlert("deploy", "prod", alert.SeverityHigh))
	h.submit(incidentAlert("svc", "prod", alert.SeverityHigh))
	h.advance(groupWait)

	messages := h.wantSent(1)
	if !strings.Contains(messages[0].subject, "shop/deployment/deploy") {
		t.Errorf("应只发源头: %+v", messages[0])
	}
	if !strings.Contains(messages[0].body, "影响范围") || !strings.Contains(messages[0].body, "shop/service/api") ||
		!strings.Contains(messages[0].body, "shop/service/db") {
		t.Errorf("源头消息应带影响范围（含没有告警的受影响事件）: %s", messages[0].body)
	}
	h.wantDeliveries("incident:svc")
	if stored := h.storedAlert("incident:svc"); stored.SuppressedBy != SuppressedByRootIncident {
		t.Errorf("受影响的告警应标记被源头抑制: %+v", stored)
	}
	if stored := h.storedAlert("incident:deploy"); stored.SuppressedBy != "" {
		t.Errorf("源头自己不被抑制: %+v", stored)
	}
}

// 源头先恢复（关联解除）而受影响的仍在持续：解除抑制，按新告警发出
func TestLinkedAlertReleasedWhenRootRecovers(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.link("svc", "deploy", "shop/service/api")
	h.submit(incidentAlert("deploy", "prod", alert.SeverityHigh))
	h.submit(incidentAlert("svc", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(1)

	h.advance(time.Minute)
	h.submit(resolved(incidentAlert("deploy", "prod", alert.SeverityHigh)))
	h.link("svc", "", "")
	h.advance(groupInterval)
	h.advance(groupWait)
	if stored := h.storedAlert("incident:svc"); stored.SuppressedBy != "" {
		t.Errorf("关联解除后应解除抑制: %+v", stored)
	}
	h.wantDeliveries("incident:svc", "firing/success")
}

// 只有关联、源头没有告警（源头只到警告、没发告警）：受影响的不被压住
func TestLinkWithoutFiringRootDoesNotSuppress(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.link("svc", "deploy", "shop/service/api")
	h.submit(incidentAlert("svc", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(1)
	h.wantDeliveries("incident:svc", "firing/success")
}
