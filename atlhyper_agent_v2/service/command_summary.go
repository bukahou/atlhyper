// atlhyper_agent_v2/service/command_summary.go
// List 响应摘要 — 将 K8s API JSON 转为 kubectl get 风格的表格
// 目的: 大幅压缩 list 结果体积，让 AI 看到全量资源列表而不被截断
package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// summarizeList 将 K8s list JSON 转为表格文本
// 返回类似:
//
//	Nodes (6):
//	NAME         STATUS  ROLES                  VERSION          AGE
//	desk-zero    Ready   control-plane,master   v1.33.5+k3s1    112d
//	...
func summarizeList(data []byte) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return string(data)
	}

	items, ok := obj["items"].([]interface{})
	if !ok {
		return string(data)
	}

	// 获取资源类型 (e.g. "NodeList" → "Node")
	kind := ""
	if k, ok := obj["kind"].(string); ok {
		kind = strings.TrimSuffix(k, "List")
	}

	// 根据 Kind 选择列定义
	columns := getColumns(kind)
	if columns == nil {
		// 未知类型: 回退到紧凑 JSON
		return string(data)
	}

	// 提取表格数据
	var rows [][]string
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = col.Extract(m)
		}
		rows = append(rows, row)
	}

	// 格式化表格
	return formatTable(kind, columns, rows)
}

// column 列定义
type column struct {
	Header  string
	Extract func(item map[string]interface{}) string
}

// getColumns 根据资源类型返回列定义
func getColumns(kind string) []column {
	switch kind {
	case "Node":
		return nodeColumns()
	case "Pod":
		return podColumns()
	case "Deployment":
		return deploymentColumns()
	case "Service":
		return serviceColumns()
	case "StatefulSet":
		return statefulSetColumns()
	case "DaemonSet":
		return daemonSetColumns()
	case "Ingress":
		return ingressColumns()
	case "Job":
		return jobColumns()
	case "CronJob":
		return cronJobColumns()
	case "PersistentVolumeClaim":
		return pvcColumns()
	case "PersistentVolume":
		return pvColumns()
	case "Namespace":
		return namespaceColumns()
	case "ConfigMap":
		return configMapColumns()
	case "Event":
		return eventColumns()
	case "ReplicaSet":
		return replicaSetColumns()
	case "HorizontalPodAutoscaler":
		return hpaColumns()
	default:
		// 通用: NAME, NAMESPACE, AGE
		return genericColumns()
	}
}

// ============================================================
// 各资源类型的列定义
// ============================================================

func nodeColumns() []column {
	return []column{
		{"NAME", extractName},
		{"STATUS", func(m map[string]interface{}) string {
			return getNodeStatus(m)
		}},
		{"ROLES", func(m map[string]interface{}) string {
			return getNodeRoles(m)
		}},
		{"VERSION", func(m map[string]interface{}) string {
			return nestedStr(m, "status", "nodeInfo", "kubeletVersion")
		}},
		{"AGE", extractAge},
	}
}

func podColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"STATUS", func(m map[string]interface{}) string {
			return getPodStatus(m)
		}},
		{"RESTARTS", func(m map[string]interface{}) string {
			return getPodRestarts(m)
		}},
		{"AGE", extractAge},
	}
}

func deploymentColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"READY", func(m map[string]interface{}) string {
			ready := nestedInt(m, "status", "readyReplicas")
			total := nestedInt(m, "spec", "replicas")
			return fmt.Sprintf("%d/%d", ready, total)
		}},
		{"UP-TO-DATE", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "status", "updatedReplicas"))
		}},
		{"AVAILABLE", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "status", "availableReplicas"))
		}},
		{"AGE", extractAge},
	}
}

func serviceColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"TYPE", func(m map[string]interface{}) string {
			return nestedStr(m, "spec", "type")
		}},
		{"CLUSTER-IP", func(m map[string]interface{}) string {
			return nestedStr(m, "spec", "clusterIP")
		}},
		{"PORTS", func(m map[string]interface{}) string {
			return getServicePorts(m)
		}},
		{"AGE", extractAge},
	}
}

func statefulSetColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"READY", func(m map[string]interface{}) string {
			ready := nestedInt(m, "status", "readyReplicas")
			total := nestedInt(m, "spec", "replicas")
			return fmt.Sprintf("%d/%d", ready, total)
		}},
		{"AGE", extractAge},
	}
}

func daemonSetColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"DESIRED", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "status", "desiredNumberScheduled"))
		}},
		{"CURRENT", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "status", "currentNumberScheduled"))
		}},
		{"READY", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "status", "numberReady"))
		}},
		{"AGE", extractAge},
	}
}

func ingressColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"HOSTS", func(m map[string]interface{}) string {
			return getIngressHosts(m)
		}},
		{"AGE", extractAge},
	}
}

func jobColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"COMPLETIONS", func(m map[string]interface{}) string {
			succeeded := nestedInt(m, "status", "succeeded")
			total := nestedInt(m, "spec", "completions")
			return fmt.Sprintf("%d/%d", succeeded, total)
		}},
		{"AGE", extractAge},
	}
}

func cronJobColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"SCHEDULE", func(m map[string]interface{}) string {
			return nestedStr(m, "spec", "schedule")
		}},
		{"SUSPEND", func(m map[string]interface{}) string {
			if nestedBool(m, "spec", "suspend") {
				return "True"
			}
			return "False"
		}},
		{"AGE", extractAge},
	}
}

func pvcColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"STATUS", func(m map[string]interface{}) string {
			return nestedStr(m, "status", "phase")
		}},
		{"VOLUME", func(m map[string]interface{}) string {
			return nestedStr(m, "spec", "volumeName")
		}},
		{"CAPACITY", func(m map[string]interface{}) string {
			return nestedStr(m, "status", "capacity", "storage")
		}},
		{"AGE", extractAge},
	}
}

func pvColumns() []column {
	return []column{
		{"NAME", extractName},
		{"CAPACITY", func(m map[string]interface{}) string {
			return nestedStr(m, "spec", "capacity", "storage")
		}},
		{"STATUS", func(m map[string]interface{}) string {
			return nestedStr(m, "status", "phase")
		}},
		{"CLAIM", func(m map[string]interface{}) string {
			ns := nestedStr(m, "spec", "claimRef", "namespace")
			name := nestedStr(m, "spec", "claimRef", "name")
			if ns == "" && name == "" {
				return "<none>"
			}
			return ns + "/" + name
		}},
		{"AGE", extractAge},
	}
}

func namespaceColumns() []column {
	return []column{
		{"NAME", extractName},
		{"STATUS", func(m map[string]interface{}) string {
			return nestedStr(m, "status", "phase")
		}},
		{"AGE", extractAge},
	}
}

func configMapColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"DATA", func(m map[string]interface{}) string {
			if data, ok := m["data"].(map[string]interface{}); ok {
				return fmt.Sprintf("%d", len(data))
			}
			return "0"
		}},
		{"AGE", extractAge},
	}
}

func eventColumns() []column {
	return []column{
		{"TYPE", func(m map[string]interface{}) string {
			return getStr(m, "type")
		}},
		{"REASON", func(m map[string]interface{}) string {
			return getStr(m, "reason")
		}},
		{"OBJECT", func(m map[string]interface{}) string {
			kind := nestedStr(m, "involvedObject", "kind")
			name := nestedStr(m, "involvedObject", "name")
			return fmt.Sprintf("%s/%s", kind, name)
		}},
		{"MESSAGE", func(m map[string]interface{}) string {
			msg := getStr(m, "message")
			if len(msg) > 80 {
				return msg[:80] + "..."
			}
			return msg
		}},
		{"AGE", func(m map[string]interface{}) string {
			ts := getStr(m, "lastTimestamp")
			if ts == "" {
				ts = nestedStr(m, "eventTime")
			}
			return formatAge(ts)
		}},
	}
}

func replicaSetColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"DESIRED", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "spec", "replicas"))
		}},
		{"CURRENT", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "status", "replicas"))
		}},
		{"READY", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "status", "readyReplicas"))
		}},
		{"AGE", extractAge},
	}
}

func hpaColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"REFERENCE", func(m map[string]interface{}) string {
			kind := nestedStr(m, "spec", "scaleTargetRef", "kind")
			name := nestedStr(m, "spec", "scaleTargetRef", "name")
			return fmt.Sprintf("%s/%s", kind, name)
		}},
		{"MINPODS", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "spec", "minReplicas"))
		}},
		{"MAXPODS", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "spec", "maxReplicas"))
		}},
		{"REPLICAS", func(m map[string]interface{}) string {
			return fmt.Sprintf("%d", nestedInt(m, "status", "currentReplicas"))
		}},
		{"AGE", extractAge},
	}
}

func genericColumns() []column {
	return []column{
		{"NAME", extractName},
		{"NAMESPACE", extractNamespace},
		{"AGE", extractAge},
	}
}

// ============================================================
// 通用字段提取器
// ============================================================

func extractName(m map[string]interface{}) string {
	return nestedStr(m, "metadata", "name")
}

func extractNamespace(m map[string]interface{}) string {
	ns := nestedStr(m, "metadata", "namespace")
	if ns == "" {
		return "<none>"
	}
	return ns
}

func extractAge(m map[string]interface{}) string {
	ts := nestedStr(m, "metadata", "creationTimestamp")
	return formatAge(ts)
}

// ============================================================
// 资源专用提取器
// ============================================================

func getNodeStatus(m map[string]interface{}) string {
	conditions := nestedSlice(m, "status", "conditions")
	for _, c := range conditions {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if getStr(cm, "type") == "Ready" {
			if getStr(cm, "status") == "True" {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func getNodeRoles(m map[string]interface{}) string {
	labels := nestedMap(m, "metadata", "labels")
	if labels == nil {
		return "<none>"
	}
	var roles []string
	for k := range labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(k, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	return strings.Join(roles, ",")
}

func getPodStatus(m map[string]interface{}) string {
	phase := nestedStr(m, "status", "phase")

	// 检查容器状态 (Waiting 的 reason 优先显示)
	containerStatuses := nestedSlice(m, "status", "containerStatuses")
	for _, cs := range containerStatuses {
		csm, ok := cs.(map[string]interface{})
		if !ok {
			continue
		}
		state, ok := csm["state"].(map[string]interface{})
		if !ok {
			continue
		}
		if waiting, ok := state["waiting"].(map[string]interface{}); ok {
			if reason := getStr(waiting, "reason"); reason != "" {
				return reason
			}
		}
		if terminated, ok := state["terminated"].(map[string]interface{}); ok {
			if reason := getStr(terminated, "reason"); reason != "" {
				return reason
			}
		}
	}

	return phase
}

func getPodRestarts(m map[string]interface{}) string {
	containerStatuses := nestedSlice(m, "status", "containerStatuses")
	total := 0
	for _, cs := range containerStatuses {
		csm, ok := cs.(map[string]interface{})
		if !ok {
			continue
		}
		if count, ok := csm["restartCount"].(float64); ok {
			total += int(count)
		}
	}
	return fmt.Sprintf("%d", total)
}

func getServicePorts(m map[string]interface{}) string {
	spec, ok := m["spec"].(map[string]interface{})
	if !ok {
		return "<none>"
	}
	ports, ok := spec["ports"].([]interface{})
	if !ok || len(ports) == 0 {
		return "<none>"
	}
	var parts []string
	for _, p := range ports {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		port := int(getFloat(pm, "port"))
		protocol := getStr(pm, "protocol")
		if protocol == "" {
			protocol = "TCP"
		}
		nodePort := int(getFloat(pm, "nodePort"))
		if nodePort > 0 {
			parts = append(parts, fmt.Sprintf("%d:%d/%s", port, nodePort, protocol))
		} else {
			parts = append(parts, fmt.Sprintf("%d/%s", port, protocol))
		}
	}
	return strings.Join(parts, ",")
}

func getIngressHosts(m map[string]interface{}) string {
	spec, ok := m["spec"].(map[string]interface{})
	if !ok {
		return "<none>"
	}
	rules, ok := spec["rules"].([]interface{})
	if !ok {
		return "<none>"
	}
	var hosts []string
	for _, r := range rules {
		rm, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if h := getStr(rm, "host"); h != "" {
			hosts = append(hosts, h)
		}
	}
	if len(hosts) == 0 {
		return "*"
	}
	return strings.Join(hosts, ",")
}

// ============================================================
// 辅助函数
// ============================================================

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

// nestedStr 安全获取嵌套字符串
func nestedStr(m map[string]interface{}, keys ...string) string {
	current := interface{}(m)
	for _, key := range keys {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = cm[key]
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

// nestedInt 安全获取嵌套整数
func nestedInt(m map[string]interface{}, keys ...string) int {
	current := interface{}(m)
	for _, key := range keys {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return 0
		}
		current = cm[key]
	}
	if f, ok := current.(float64); ok {
		return int(f)
	}
	return 0
}

// nestedBool 安全获取嵌套布尔值
func nestedBool(m map[string]interface{}, keys ...string) bool {
	current := interface{}(m)
	for _, key := range keys {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return false
		}
		current = cm[key]
	}
	if b, ok := current.(bool); ok {
		return b
	}
	return false
}

// nestedSlice 安全获取嵌套切片
func nestedSlice(m map[string]interface{}, keys ...string) []interface{} {
	current := interface{}(m)
	for _, key := range keys {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = cm[key]
	}
	if s, ok := current.([]interface{}); ok {
		return s
	}
	return nil
}

// nestedMap 安全获取嵌套 map
func nestedMap(m map[string]interface{}, keys ...string) map[string]interface{} {
	current := interface{}(m)
	for _, key := range keys {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = cm[key]
	}
	if cm, ok := current.(map[string]interface{}); ok {
		return cm
	}
	return nil
}

// formatAge 将时间戳格式化为年龄字符串 (如 "5d", "3h", "12m")
func formatAge(timestamp string) string {
	if timestamp == "" {
		return "<unknown>"
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "<unknown>"
	}
	d := time.Since(t)
	if d < 0 {
		return "0s"
	}

	days := int(math.Floor(d.Hours() / 24))
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(math.Floor(d.Hours()))
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(math.Floor(d.Minutes()))
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// formatTable 格式化对齐表格
func formatTable(kind string, columns []column, rows [][]string) string {
	if len(rows) == 0 {
		return fmt.Sprintf("%s: 0 items", kind)
	}

	// 计算每列最大宽度
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col.Header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder

	// 标题行
	sb.WriteString(fmt.Sprintf("%s (%d):\n", kind, len(rows)))

	// 表头
	for i, col := range columns {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(fmt.Sprintf("%-*s", widths[i], col.Header))
	}
	sb.WriteString("\n")

	// 数据行
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteString("  ")
			}
			sb.WriteString(fmt.Sprintf("%-*s", widths[i], cell))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
