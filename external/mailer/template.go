// 📄 mailer/template/template.go
//
// ✨ 功能说明：
//     使用 HTML 模板渲染告警邮件内容，支持多条告警聚合展示（按资源种类、命名空间、节点等分类）。
// =================================================================================

package mailer

import (
	"NeuroController/internal/types"
	"bytes"
	"fmt"
	"html/template"
)

// AlertItem 表示单条告警信息

// RenderAlertTemplate 渲染 HTML 告警聚合模板
func RenderAlertTemplate(data types.AlertGroupData) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>システム異常アラート</title>
    <style>
        body {
            font-family: "メイリオ", sans-serif;
            background-color: #f4f4f4;
            padding: 20px;
        }
        .container {
            max-width: 800px;
            margin: auto;
            background-color: #ffffff;
            padding: 20px;
            border-left: 6px solid #ff4d4f;
            box-shadow: 0 2px 8px rgba(0,0,0,0.05);
        }
        h2 {
            color: #ff4d4f;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 15px;
        }
        th, td {
            padding: 10px;
            border: 1px solid #ddd;
            text-align: left;
        }
        th {
            background-color: #ffeded;
        }
        .footer {
            margin-top: 30px;
            font-size: 12px;
            color: #999;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <h2>⚠️ システム異常アラート（全 {{.AlertCount}} 件）</h2>
        <p><strong>影響ノード：</strong><br>
        {{range .NodeList}}{{.}}<br>{{end}}
        </p>
        <p><strong>対象ネームスペース：</strong> {{range .NamespaceList}}{{.}} {{end}}</p>

        <table>
            <thead>
                <tr>
                    <th>リソース種別</th>
                    <th>名前</th>
                    <th>ネームスペース</th>
                    <th>ノード</th> 
                    <th>重要度</th>
                    <th>理由</th>
                    <th>発生時刻</th>
                    <th>説明</th>
                </tr>
            </thead>
            <tbody>
                {{range .Alerts}}
                <tr>
                    <td>{{.Kind}}</td>
                    <td>{{.Name}}</td>
                    <td>{{.Namespace}}</td>
                    <td>{{.Node}}</td> 
                    <td>{{.Severity}}</td>
                    <td>{{.Reason}}</td>
                    <td>{{.Time}}</td>
                    <td>{{.Message}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <div class="footer">
            ※このメールはシステムより自動送信されています。返信しないでください。
        </div>
    </div>
</body>
</html>
`

	t, err := template.New("alert_group").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return buf.String(), nil
}
