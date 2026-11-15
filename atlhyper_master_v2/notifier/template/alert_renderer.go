// atlhyper_master_v2/notifier/template/alert_renderer.go
// 告警消息渲染：单条 / 分组 × Slack / 邮件
package template

import (
	"bytes"
	"embed"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
)

//go:embed templates/alert_*.tmpl
var alertTemplateFS embed.FS

const (
	// slackBodyLimit Slack 一个文本块的长度上限是 3000 字符，留一点余量；超了整条消息会被拒收
	slackBodyLimit = 2900
	// groupItemLimit 一条分组消息里最多逐条列出多少条告警，其余只报数量
	groupItemLimit = 20
	// incidentDetailPath 事件详情页在前端的路径（事件号作为查询参数）
	incidentDetailPath = "/aiops/incidents?incident="
)

// Item 一条通知里的一条告警，以及这次要告诉收件人的变化。
// Kind 与告警自身的 Kind 可以不同：重复提醒时告警的 Kind 仍是它最近一次的变化
type Item struct {
	Alert alert.Alert
	Kind  alert.Kind
}

// AlertRenderer 告警消息渲染器
type AlertRenderer struct {
	publicURL string // Master 的对外地址；空 = 消息里不带页面链接
	templates map[string]*template.Template
}

// messageView 模板的输入：一条消息
type messageView struct {
	Title   string
	Items   []itemView
	Omitted int // 分组过大时没有逐条列出的条数
}

// itemView 模板的输入：消息里的一条告警
type itemView struct {
	Head   string      // 严重度 · 主体 · 摘要
	Fields []fieldView // 逐行展示的字段（单条消息用）
	Brief  string      // 压成一行的要点（分组消息用）
	Link   string      // 事件详情页；空 = 不带
}

type fieldView struct {
	Label string
	Value string
}

// NewAlertRenderer 创建告警消息渲染器。publicURL 为空时消息里不带页面链接
func NewAlertRenderer(publicURL string) (*AlertRenderer, error) {
	renderer := &AlertRenderer{
		publicURL: strings.TrimRight(strings.TrimSpace(publicURL), "/"),
		templates: make(map[string]*template.Template),
	}
	for _, layout := range []string{"single", "group"} {
		for _, receiverType := range []string{database.NotifyReceiverSlack, database.NotifyReceiverEmail} {
			name := fmt.Sprintf("alert_%s_%s", layout, receiverType)
			parsed, err := template.ParseFS(alertTemplateFS, "templates/"+name+".tmpl")
			if err != nil {
				return nil, fmt.Errorf("解析模板 %s 失败: %w", name, err)
			}
			renderer.templates[layout+"_"+receiverType] = parsed
		}
	}
	return renderer, nil
}

// Render 把一批告警渲染成发给某类接收人的一条消息。一条用单条模板，多条用分组模板
func (r *AlertRenderer) Render(receiverType string, items []Item, now time.Time) (*channel.Message, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("没有可渲染的告警")
	}
	layout := "single"
	if len(items) > 1 {
		layout = "group"
	}
	tmpl, ok := r.templates[layout+"_"+receiverType]
	if !ok {
		return nil, fmt.Errorf("没有适用于接收人类型 %q 的模板", receiverType)
	}

	// 告警内容来自集群，不可信：发往 Slack 的每一段动态文字都先转义
	escape := func(text string) string { return text }
	format := "text"
	if receiverType == database.NotifyReceiverSlack {
		escape, format = escapeSlack, "markdown"
	}

	view := messageView{Title: escape(messageTitle(items))}
	listed := items
	if len(listed) > groupItemLimit {
		view.Omitted = len(listed) - groupItemLimit
		listed = listed[:groupItemLimit]
	}
	for _, item := range listed {
		view.Items = append(view.Items, r.itemView(item, now, escape))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, view); err != nil {
		return nil, fmt.Errorf("渲染模板失败: %w", err)
	}
	body := strings.TrimSpace(buf.String())
	if receiverType == database.NotifyReceiverSlack {
		body = truncateRunes(body, slackBodyLimit)
	}
	// 标题也按接收人类型转义：Slack 把它当作消息的纯文本摘要，同样会解析控制序列
	return &channel.Message{Subject: view.Title, Body: body, Format: format}, nil
}

// RenderTest 测试消息：只用来验证接收人配置能否送达，不对应任何告警
func (r *AlertRenderer) RenderTest(receiverName string, now time.Time) *channel.Message {
	return &channel.Message{
		Subject: "AtlHyper 测试通知",
		Body: fmt.Sprintf("这是一条测试消息，用于验证接收人「%s」的配置是否正确。\n发送时间: %s",
			escapeSlack(receiverName), now.UTC().Format("2006-01-02 15:04:05 MST")),
		Format: "text",
	}
}

// messageTitle 单条：[集群] 严重度 · 主体 · 一句话摘要；分组：[集群] N 条告警（最高严重度）
func messageTitle(items []Item) string {
	first := items[0].Alert
	if len(items) == 1 {
		return fmt.Sprintf("[%s] %s", first.ClusterID, itemHead(&first))
	}
	highest := first.Severity
	for _, item := range items[1:] {
		highest = alert.MaxSeverity(highest, item.Alert.Severity)
	}
	return fmt.Sprintf("[%s] %d 条告警（最高%s）", first.ClusterID, len(items), SeverityText(highest))
}

// itemHead 严重度 · 主体 · 摘要。没有主体的告警（如心跳）用告警标题占位
func itemHead(a *alert.Alert) string {
	subject := a.Label(alert.LabelSubject)
	if subject == "" {
		subject = a.Title
	}
	parts := []string{SeverityText(a.Severity), subject}
	if a.Summary != "" {
		parts = append(parts, a.Summary)
	}
	return strings.Join(parts, " · ")
}

func (r *AlertRenderer) itemView(item Item, now time.Time, escape func(string) string) itemView {
	a := &item.Alert
	fields := []fieldView{{"变化", KindText(item.Kind)}}
	add := func(label, value string) {
		if value != "" {
			fields = append(fields, fieldView{label, escape(value)})
		}
	}
	add("主体", a.Label(alert.LabelSubject))
	add("根因实体", a.Detail(alert.DetailRootCause))
	add("成员数", a.Detail(alert.DetailMemberCount))
	add("峰值风险", a.Detail(alert.DetailPeakRisk))
	add("持续时间", formatDuration(alertDuration(a, now)))
	if recurrence := a.Detail(alert.DetailRecurrence); recurrence != "0" {
		add("复发次数", recurrence)
	}
	if item.Kind == alert.KindFlapping {
		add("近 24 小时发生次数", a.Detail(alert.DetailOccurrences24h))
	}
	add("最后心跳", a.Detail(alert.DetailLastHeartbeat))
	add("事件原因", a.Detail(alert.DetailEventReason))
	add("事件次数", a.Detail(alert.DetailEventCount))
	add("开始时间", a.StartsAt.UTC().Format("2006-01-02 15:04:05 MST"))

	// 分组消息里每条只占一行：变化、根因、峰值风险、持续时间
	var brief []string
	for _, field := range fields {
		switch field.Label {
		case "变化", "根因实体", "峰值风险", "持续时间":
			brief = append(brief, field.Label+": "+field.Value)
		}
	}
	return itemView{
		Head:   escape(itemHead(a)),
		Fields: fields,
		Brief:  strings.Join(brief, " · "),
		Link:   r.incidentLink(a),
	}
}

// incidentLink 事件详情页的链接；没配对外地址、或告警不对应事件时为空
func (r *AlertRenderer) incidentLink(a *alert.Alert) string {
	incidentID := a.Detail(alert.DetailIncidentID)
	if r.publicURL == "" || incidentID == "" {
		return ""
	}
	return r.publicURL + incidentDetailPath + url.QueryEscape(incidentID)
}

// alertDuration 已结束的告警算到结束时刻，持续中的算到现在
func alertDuration(a *alert.Alert, now time.Time) time.Duration {
	if a.EndsAt != nil {
		return a.EndsAt.Sub(a.StartsAt)
	}
	return now.Sub(a.StartsAt)
}

// KindText 变化的中文说法
func KindText(kind alert.Kind) string {
	switch kind {
	case alert.KindFiring:
		return "触发"
	case alert.KindEscalated:
		return "升级"
	case alert.KindRecurred:
		return "复发"
	case alert.KindFlapping:
		return "反复"
	case alert.KindResolved:
		return "恢复"
	case alert.KindRepeat:
		return "重复提醒"
	default:
		return string(kind)
	}
}

// SeverityText 严重度的中文说法
func SeverityText(severity alert.Severity) string {
	switch severity {
	case alert.SeverityLow:
		return "低"
	case alert.SeverityMedium:
		return "中"
	case alert.SeverityHigh:
		return "高"
	case alert.SeverityCritical:
		return "严重"
	default:
		return string(severity)
	}
}

// formatDuration 把时长写成「2 天 1 小时」「1 小时 5 分钟」「5 分钟」；最多两段
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "不到 1 分钟"
	}
	days := int(d / (24 * time.Hour))
	hours := int(d/time.Hour) % 24
	minutes := int(d/time.Minute) % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d 天", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d 小时", hours))
	}
	if minutes > 0 && days == 0 {
		parts = append(parts, fmt.Sprintf("%d 分钟", minutes))
	}
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, " ")
}

// escapeSlack Slack 文本里 & < > 是控制字符（<!channel>、<链接|文字>），按官方约定转义
func escapeSlack(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}

// truncateRunes 按字符数截断，被截断时以省略号结尾
func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
}
