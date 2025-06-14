// 📄 mailer/template/template.go
//
// ✨ 功能说明：
//     使用 HTML 模板渲染告警邮件内容，支持多条告警聚合展示（按资源种类、命名空间、节点等分类）。
// =================================================================================

package mailer

import (
	"bytes"
	"fmt"
	"html/template"
)

// AlertItem 表示单条告警信息
type AlertItem struct {
	Kind      string
	Name      string
	Namespace string
	Node      string
	Severity  string
	Reason    string
	Message   string
	Time      string
}

// AlertGroupData 聚合告警模板数据结构
type AlertGroupData struct {
	Title         string
	NodeList      []string
	NamespaceList []string
	AlertCount    int
	Alerts        []AlertItem
}

// RenderAlertTemplate 渲染 HTML 告警聚合模板
func RenderAlertTemplate(data AlertGroupData) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>系统异常告警</title>
    <style>
        body {
            font-family: "微软雅黑", sans-serif;
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
        <h2>⚠️ 系统异常告警（共 {{.AlertCount}} 条）</h2>
        <p><strong>涉及节点：</strong> {{range .NodeList}}{{.}} {{end}}</p>
        <p><strong>涉及命名空间：</strong> {{range .NamespaceList}}{{.}} {{end}}</p>

        <table>
            <thead>
                <tr>
                    <th>资源类型</th>
                    <th>名称</th>
                    <th>命名空间</th>
                    <th>节点</th> 
                    <th>等级</th>
                    <th>原因</th>
                    <th>时间</th>
                    <th>描述</th>
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
            此邮件由系统自动发出，请勿回复。
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
