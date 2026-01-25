// atlhyper_master/gateway/integration/mailer/template_html.go
// 邮件 HTML 模板定义
package mailer

const emailTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>系统告警通知</title>
</head>
<body style="margin:0;padding:0;background-color:#F3F4F6;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
    <table width="100%" cellpadding="0" cellspacing="0" style="background-color:#F3F4F6;padding:20px 0;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" style="background-color:#FFFFFF;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
                    <!-- 头部 -->
                    <tr>
                        <td style="background:linear-gradient(135deg,#1E40AF 0%,#3B82F6 100%);padding:24px 32px;border-radius:8px 8px 0 0;">
                            <h1 style="margin:0;color:#FFFFFF;font-size:20px;font-weight:600;">
                                🚨 系统告警通知
                            </h1>
                            <p style="margin:8px 0 0;color:#BFDBFE;font-size:14px;">
                                共检测到 {{.AlertCount}} 条告警
                            </p>
                        </td>
                    </tr>

                    <!-- 统计卡片 -->
                    <tr>
                        <td style="padding:24px 32px 16px;">
                            <table width="100%" cellpadding="0" cellspacing="0">
                                <tr>
                                    {{if gt .CriticalCount 0}}
                                    <td width="33%" style="padding:0 8px 0 0;">
                                        <div style="background-color:#FEE2E2;border-radius:8px;padding:16px;text-align:center;">
                                            <div style="font-size:28px;font-weight:700;color:#DC2626;">{{.CriticalCount}}</div>
                                            <div style="font-size:12px;color:#991B1B;margin-top:4px;">🔴 严重</div>
                                        </div>
                                    </td>
                                    {{end}}
                                    {{if gt .WarningCount 0}}
                                    <td width="33%" style="padding:0 8px;">
                                        <div style="background-color:#FEF3C7;border-radius:8px;padding:16px;text-align:center;">
                                            <div style="font-size:28px;font-weight:700;color:#D97706;">{{.WarningCount}}</div>
                                            <div style="font-size:12px;color:#92400E;margin-top:4px;">🟠 警告</div>
                                        </div>
                                    </td>
                                    {{end}}
                                    {{if gt .InfoCount 0}}
                                    <td width="33%" style="padding:0 0 0 8px;">
                                        <div style="background-color:#DBEAFE;border-radius:8px;padding:16px;text-align:center;">
                                            <div style="font-size:28px;font-weight:700;color:#2563EB;">{{.InfoCount}}</div>
                                            <div style="font-size:12px;color:#1E40AF;margin-top:4px;">🔵 信息</div>
                                        </div>
                                    </td>
                                    {{end}}
                                </tr>
                            </table>
                        </td>
                    </tr>

                    <!-- 摘要信息 -->
                    <tr>
                        <td style="padding:0 32px 24px;">
                            <table width="100%" style="background-color:#F9FAFB;border-radius:8px;padding:16px;" cellpadding="8">
                                <tr>
                                    <td style="color:#6B7280;font-size:13px;width:100px;">🏷️ 集群</td>
                                    <td style="color:#111827;font-size:13px;font-weight:500;">{{.ClusterStr}}</td>
                                </tr>
                                <tr>
                                    <td style="color:#6B7280;font-size:13px;">📁 命名空间</td>
                                    <td style="color:#111827;font-size:13px;font-weight:500;">{{.NamespaceStr}}</td>
                                </tr>
                                <tr>
                                    <td style="color:#6B7280;font-size:13px;">🖥️ 节点</td>
                                    <td style="color:#111827;font-size:13px;font-weight:500;">{{.NodeStr}}</td>
                                </tr>
                            </table>
                        </td>
                    </tr>

                    <!-- 告警明细标题 -->
                    <tr>
                        <td style="padding:0 32px 16px;">
                            <h2 style="margin:0;font-size:16px;color:#374151;border-bottom:2px solid #E5E7EB;padding-bottom:8px;">
                                📋 告警明细
                            </h2>
                        </td>
                    </tr>

                    <!-- 告警列表 -->
                    <tr>
                        <td style="padding:0 32px 24px;">
                            {{range .Alerts}}
                            <div style="background-color:{{severityBg .Severity}};border-left:4px solid {{severityColor .Severity}};border-radius:0 8px 8px 0;padding:16px;margin-bottom:12px;">
                                <div style="display:flex;align-items:center;margin-bottom:8px;">
                                    <span style="font-size:14px;">{{severityIcon .Severity}}</span>
                                    <span style="font-weight:600;color:#111827;margin-left:8px;">{{.Kind}}/{{.Name}}</span>
                                    <span style="background-color:{{severityColor .Severity}};color:#FFFFFF;font-size:11px;padding:2px 8px;border-radius:10px;margin-left:auto;">{{.Severity}}</span>
                                </div>
                                <table width="100%" style="font-size:13px;color:#4B5563;" cellpadding="4">
                                    <tr>
                                        <td width="80" style="color:#6B7280;">集群</td>
                                        <td>{{.ClusterID}}</td>
                                        <td width="80" style="color:#6B7280;">命名空间</td>
                                        <td>{{.Namespace}}</td>
                                    </tr>
                                    <tr>
                                        <td style="color:#6B7280;">节点</td>
                                        <td>{{.Node}}</td>
                                        <td style="color:#6B7280;">时间</td>
                                        <td>{{.Time}}</td>
                                    </tr>
                                    <tr>
                                        <td style="color:#6B7280;">原因</td>
                                        <td colspan="3" style="font-weight:500;">{{.Reason}}</td>
                                    </tr>
                                    {{if .Message}}
                                    <tr>
                                        <td style="color:#6B7280;vertical-align:top;">详情</td>
                                        <td colspan="3" style="word-break:break-all;">{{truncate .Message 300}}</td>
                                    </tr>
                                    {{end}}
                                </table>
                            </div>
                            {{end}}
                        </td>
                    </tr>

                    <!-- 页脚 -->
                    <tr>
                        <td style="background-color:#F9FAFB;padding:20px 32px;border-radius:0 0 8px 8px;text-align:center;">
                            <p style="margin:0;font-size:12px;color:#9CA3AF;">
                                此邮件由 AtlHyper 监控系统自动发送，请勿回复
                            </p>
                            <p style="margin:8px 0 0;font-size:11px;color:#D1D5DB;">
                                © 2024 AtlHyper - Kubernetes 智能运维平台
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`
