package types

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
