// atlhyper_master_v2/notifier/template/renderer.go
// 模板渲染器
package template

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
	"time"

	"AtlHyper/atlhyper_master_v2/notifier/channel"
	"AtlHyper/atlhyper_master_v2/notifier/enrich"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// AlertData 模板数据
type AlertData struct {
	// 基础信息
	Title     string
	Message   string
	Severity  string
	Source    string
	ClusterID string
	Resource  string
	Reason    string
	Timestamp time.Time
	TimeStr   string // 格式化后的时间

	// 附加字段
	Fields map[string]string

	// 丰富数据
	Enriched *enrich.EnrichedData

	// 渠道特定
	SeverityEmoji string
}

// Renderer 模板渲染器
type Renderer struct {
	templates map[string]*template.Template
}

// NewRenderer 创建渲染器
func NewRenderer() (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
	}

	// 加载所有模板
	templateNames := []string{
		"heartbeat_offline",
		"heartbeat_recovery",
		"k8s_event",
	}

	for _, name := range templateNames {
		// Slack 模板
		slackTmpl, err := template.ParseFS(templateFS, fmt.Sprintf("templates/%s_slack.tmpl", name))
		if err != nil {
			return nil, fmt.Errorf("parse %s_slack.tmpl: %w", name, err)
		}
		r.templates[name+"_slack"] = slackTmpl

		// Email 模板
		emailTmpl, err := template.ParseFS(templateFS, fmt.Sprintf("templates/%s_email.tmpl", name))
		if err != nil {
			return nil, fmt.Errorf("parse %s_email.tmpl: %w", name, err)
		}
		r.templates[name+"_email"] = emailTmpl
	}

	return r, nil
}

// Render 渲染告警消息
// templateName: heartbeat_offline, heartbeat_recovery, k8s_event
// channelType: slack, email
func (r *Renderer) Render(templateName, channelType string, data *AlertData) (*channel.Message, error) {
	// 补充数据
	data.TimeStr = data.Timestamp.Format("2006-01-02 15:04:05 MST")
	data.SeverityEmoji = severityEmoji(data.Severity)

	// 查找模板
	key := templateName + "_" + channelType
	tmpl, ok := r.templates[key]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", key)
	}

	// 渲染
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	// 构建消息
	msg := &channel.Message{
		Subject: data.Title,
		Body:    buf.String(),
		Format:  "markdown",
	}

	if channelType == "email" {
		msg.Format = "text"
	}

	return msg, nil
}

// severityEmoji 获取级别对应的 emoji
func severityEmoji(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "warning":
		return "🟡"
	default:
		return "🟢"
	}
}
