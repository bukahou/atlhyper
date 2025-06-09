package diagnosis

import (
	"time"

	"NeuroController/internal/utils/abnormal"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ✅ 日志事件统一结构
type LogEvent struct {
	Timestamp  time.Time
	Kind       string // Pod / Node / ...
	Namespace  string
	Name       string
	ReasonCode string
	Category   string
	Severity   string
	Message    string
}

// ✅ 全局收集池
var eventPool = make([]LogEvent, 0)

// 封装日志事件写入池（仅供本包内部使用）
func appendToEventPool(event LogEvent) {
	mu.Lock()
	defer mu.Unlock()
	eventPool = append(eventPool, event)
}

// ✅ 收集器主接口（供外部调用）
// PodWatcher 中调用此函数，无需再管内部细节
func CollectPodAbnormalEvent(pod corev1.Pod, reason *abnormal.PodAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Pod",
		Namespace:  pod.Namespace,
		Name:       pod.Name,
		ReasonCode: reason.Code,
		Category:   reason.Category,
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 收到 Pod 异常事件：%s/%s → %s（%s）\n",
	// 	pod.Namespace, pod.Name, reason.Code, reason.Message)
}

// ✅ 收集器：Node 异常事件
func CollectNodeAbnormalEvent(node corev1.Node, reason *abnormal.NodeAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Node",
		Namespace:  "", // Node 无命名空间
		Name:       node.Name,
		ReasonCode: reason.Code,
		Category:   reason.Category,
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 收到 Node 异常事件：%s → %s（%s）\n",
	// 	node.Name, reason.Code, reason.Message)
}

// ✅ 收集器：Event 异常事件
func CollectEventAbnormalEvent(ev corev1.Event, reason *abnormal.EventAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       ev.InvolvedObject.Kind,
		Namespace:  ev.InvolvedObject.Namespace,
		Name:       ev.InvolvedObject.Name,
		ReasonCode: reason.Code,
		Category:   "Event", // 可自定义为分类
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 收到 Event 异常事件：%s/%s（%s）→ %s\n",
	// 	ev.InvolvedObject.Namespace, ev.InvolvedObject.Name, ev.InvolvedObject.Kind, reason.Message)

}

// ✅ 收集器：Endpoint 异常事件
func CollectEndpointAbnormalEvent(ep corev1.Endpoints, reason *abnormal.EndpointAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Endpoints",
		Namespace:  ep.Namespace,
		Name:       ep.Name,
		ReasonCode: reason.Code,
		Category:   "Endpoint", // 可选：用于后续聚合分析分类
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 收到 Endpoint 异常事件：%s/%s → %s（%s）\n",
	// 	ep.Namespace, ep.Name, reason.Code, reason.Message)
}

// ✅ 收集器：Deployment 异常事件
func CollectDeploymentAbnormalEvent(deploy appsv1.Deployment, reason *abnormal.DeploymentAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Deployment",
		Namespace:  deploy.Namespace,
		Name:       deploy.Name,
		ReasonCode: reason.Code,
		Category:   reason.Category,
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 收到 Deployment 异常事件：%s/%s → %s（%s）\n",
	// 	deploy.Namespace, deploy.Name, reason.Code, reason.Message)
}

// ✅ 收集器：Service 异常事件
func CollectServiceAbnormalEvent(svc corev1.Service, reason *abnormal.ServiceAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Service",
		Namespace:  svc.Namespace,
		Name:       svc.Name,
		ReasonCode: reason.Code,
		Category:   "Warning", // 如需扩展可在 reason 中添加字段
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 收到 Service 异常事件：%s/%s → %s（%s）\n",
	// 	svc.Namespace, svc.Name, reason.Code, reason.Message)
}
