package model

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