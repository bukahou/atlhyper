// model/integration/alert.go
// 告警推送模型（Master → Slack/邮件等第三方）
package integration

type LightweightAlertStub struct {
	Title   string
	Data    AlertGroupData
	Display bool
}

// AlertGroupData 聚合告警模板数据结构
type AlertGroupData struct {
	Title         string
	ClusterID     []string
	NodeList      []string
	NamespaceList []string
	AlertCount    int
	Alerts        []AlertItem
}

type AlertItem struct {
	ClusterID string
	Kind      string
	Name      string
	Namespace string
	Node      string
	Severity  string
	Reason    string
	Message   string
	Time      string
}
