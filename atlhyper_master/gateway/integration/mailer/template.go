// atlhyper_master/gateway/integration/mailer/template.go
// 邮件告警 HTML 模板渲染
package mailer

import (
	"AtlHyper/model/integration"
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// RenderAlertTemplate 渲染 HTML 告警聚合模板
func RenderAlertTemplate(data integration.AlertGroupData) (string, error) {
	// 预处理数据
	viewData := prepareViewData(data)

	t, err := template.New("alert").Funcs(template.FuncMap{
		"severityColor": severityColor,
		"severityBg":    severityBg,
		"severityIcon":  severityIcon,
		"truncate":      truncateText,
	}).Parse(emailTemplate)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, viewData); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return buf.String(), nil
}

// AlertViewData 视图数据
type AlertViewData struct {
	integration.AlertGroupData
	CriticalCount int
	WarningCount  int
	InfoCount     int
	ClusterStr    string
	NamespaceStr  string
	NodeStr       string
}

// prepareViewData 预处理视图数据
func prepareViewData(data integration.AlertGroupData) AlertViewData {
	view := AlertViewData{AlertGroupData: data}

	// 统计各级别数量
	for _, alert := range data.Alerts {
		switch strings.ToLower(alert.Severity) {
		case "critical":
			view.CriticalCount++
		case "warning":
			view.WarningCount++
		default:
			view.InfoCount++
		}
	}

	// 格式化列表
	view.ClusterStr = formatListStr(data.ClusterID, 5)
	view.NamespaceStr = formatListStr(data.NamespaceList, 8)
	view.NodeStr = formatListStr(data.NodeList, 5)

	return view
}

func formatListStr(items []string, max int) string {
	if len(items) == 0 {
		return "-"
	}
	if len(items) <= max {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:max], ", ") + fmt.Sprintf(" (+%d)", len(items)-max)
}

func severityColor(sev string) string {
	switch strings.ToLower(sev) {
	case "critical":
		return "#DC2626"
	case "warning":
		return "#D97706"
	default:
		return "#2563EB"
	}
}

func severityBg(sev string) string {
	switch strings.ToLower(sev) {
	case "critical":
		return "#FEE2E2"
	case "warning":
		return "#FEF3C7"
	default:
		return "#DBEAFE"
	}
}

func severityIcon(sev string) string {
	switch strings.ToLower(sev) {
	case "critical":
		return "🔴"
	case "warning":
		return "🟠"
	default:
		return "🔵"
	}
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
