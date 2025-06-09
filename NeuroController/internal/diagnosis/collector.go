package diagnosis

import (
	"time"

	"NeuroController/internal/utils/abnormal"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// =======================================================================================
// 📄 diagnosis/collector.go
//
// ✨ Description:
//     Provides a unified interface for collecting abnormal events from various
//     Kubernetes resources (Pod, Node, Event, Endpoint, etc.).
//
// 📦 Responsibilities:
//     - Define the LogEvent structure for consistent event representation
//     - Provide entry points for each resource type to report abnormal states
//     - Append events to the internal event pool for further processing
// =======================================================================================

// ✅ Unified structure for log events
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

// ✅ Global in-memory event pool (raw collected events)
var eventPool = make([]LogEvent, 0)

// Internal utility to append a log event (thread-safe, internal only)
func appendToEventPool(event LogEvent) {
	mu.Lock()
	defer mu.Unlock()
	eventPool = append(eventPool, event)
}

// ✅ Collector for abnormal Pod events
// Called by PodWatcher; encapsulates all internal logic
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

	// fmt.Printf("📥 Received Pod abnormal event: %s/%s → %s (%s)\n",
	// 	pod.Namespace, pod.Name, reason.Code, reason.Message)
}

// ✅ Collector for abnormal Node events
func CollectNodeAbnormalEvent(node corev1.Node, reason *abnormal.NodeAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Node",
		Namespace:  "", // Nodes have no namespace
		Name:       node.Name,
		ReasonCode: reason.Code,
		Category:   reason.Category,
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 Received Node abnormal event: %s → %s (%s)\n",
	// 	node.Name, reason.Code, reason.Message)
}

// ✅ Collector for abnormal corev1.Event objects
func CollectEventAbnormalEvent(ev corev1.Event, reason *abnormal.EventAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       ev.InvolvedObject.Kind,
		Namespace:  ev.InvolvedObject.Namespace,
		Name:       ev.InvolvedObject.Name,
		ReasonCode: reason.Code,
		Category:   "Event", // Categorization for analysis
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 Received Event abnormal event: %s/%s (%s) → %s\n",
	// 	ev.InvolvedObject.Namespace, ev.InvolvedObject.Name, ev.InvolvedObject.Kind, reason.Message)
}

// ✅ Collector for abnormal Endpoint events
func CollectEndpointAbnormalEvent(ep corev1.Endpoints, reason *abnormal.EndpointAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Endpoints",
		Namespace:  ep.Namespace,
		Name:       ep.Name,
		ReasonCode: reason.Code,
		Category:   "Endpoint", // Used for grouping and filtering
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 Received Endpoint abnormal event: %s/%s → %s (%s)\n",
	// 	ep.Namespace, ep.Name, reason.Code, reason.Message)
}

// ✅ Collector for abnormal Deployment events
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

	// fmt.Printf("📥 Received Deployment abnormal event: %s/%s → %s (%s)\n",
	// 	deploy.Namespace, deploy.Name, reason.Code, reason.Message)
}

// ✅ Collector for abnormal Service events
func CollectServiceAbnormalEvent(svc corev1.Service, reason *abnormal.ServiceAbnormalReason) {
	event := LogEvent{
		Timestamp:  time.Now(),
		Kind:       "Service",
		Namespace:  svc.Namespace,
		Name:       svc.Name,
		ReasonCode: reason.Code,
		Category:   "Warning", // Optional: can extend to include this in the reason struct
		Severity:   reason.Severity,
		Message:    reason.Message,
	}
	appendToEventPool(event)

	// fmt.Printf("📥 Received Service abnormal event: %s/%s → %s (%s)\n",
	// 	svc.Namespace, svc.Name, reason.Code, reason.Message)
}
