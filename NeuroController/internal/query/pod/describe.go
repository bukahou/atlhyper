package pod

import (
	"context"
	"fmt"

	"NeuroController/internal/utils"

	operatorpod "NeuroController/internal/operator/pod"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodDescribeInfo struct {
	Pod     *corev1.Pod     `json:"pod"`
	Events  []corev1.Event  `json:"events"`
	Usage   *PodUsageInfo   `json:"usage,omitempty"`
	Service *corev1.Service `json:"service,omitempty"`
	Message string          `json:"message,omitempty"`
	Logs    string          `json:"logs,omitempty"`
}

type PodUsageInfo struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// === 单位格式化函数 ===

func formatCPU(q *resource.Quantity) string {
	if q == nil {
		return "N/A"
	}
	core := q.AsApproximateFloat64()
	return fmt.Sprintf("%.2f core", core)
}

func formatMemory(q *resource.Quantity) string {
	if q == nil {
		return "N/A"
	}
	mi := float64(q.Value()) / 1024.0 / 1024.0
	return fmt.Sprintf("%.1f Mi", mi)
}

// === Service 选择器匹配 ===

func selectorMatches(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// === 查找对应 Service ===

func findServiceForPod(ctx context.Context, pod *corev1.Pod) (*corev1.Service, error) {
	client := utils.GetCoreClient()
	ns := pod.Namespace

	// 获取 ReplicaSet
	var rsName string
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			rsName = owner.Name
			break
		}
	}
	if rsName == "" {
		return nil, nil
	}
	rs, err := client.AppsV1().ReplicaSets(ns).Get(ctx, rsName, metav1.GetOptions{})
	if err != nil {
		return nil, nil
	}

	// 获取 Deployment
	var deployName string
	for _, owner := range rs.OwnerReferences {
		if owner.Kind == "Deployment" {
			deployName = owner.Name
			break
		}
	}
	if deployName == "" {
		return nil, nil
	}
	deploy, err := client.AppsV1().Deployments(ns).Get(ctx, deployName, metav1.GetOptions{})
	if err != nil {
		return nil, nil
	}

	// 匹配 Service
	svcs, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil
	}
	for _, svc := range svcs.Items {
		if selectorMatches(svc.Spec.Selector, deploy.Spec.Template.Labels) {
			return &svc, nil
		}
	}
	return nil, nil
}

// === 主函数 ===

// func GetPodDescribeInfo(ctx context.Context, namespace, name string) (*PodDescribeInfo, error) {
// 	client := utils.GetCoreClient()

// 	// 获取 Pod
// 	pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("无法获取 Pod：%w", err)
// 	}

// 	// 获取 Events
// 	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
// 		FieldSelector: fmt.Sprintf("involvedObject.name=%s", name),
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("无法获取 Event：%w", err)
// 	}

// 	// 获取 Metrics
// 	var usage *PodUsageInfo
// 	if utils.HasMetricsServer() {
// 		if metricsClient := utils.GetMetricsClient(); metricsClient != nil {
// 			metric, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, name, metav1.GetOptions{})
// 			if err == nil && len(metric.Containers) > 0 {
// 				cpu := formatCPU(metric.Containers[0].Usage.Cpu())
// 				mem := formatMemory(metric.Containers[0].Usage.Memory())
// 				usage = &PodUsageInfo{CPU: cpu, Memory: mem}
// 			}
// 		}
// 	}

// 	// 查找 Service
// 	service, _ := findServiceForPod(ctx, pod)

//		return &PodDescribeInfo{
//			Pod:     pod,
//			Events:  events.Items,
//			Usage:   usage,
//			Service: service,
//		}, nil
//	}
func GetPodDescribeInfo(ctx context.Context, namespace, name string) (*PodDescribeInfo, error) {
	client := utils.GetCoreClient()

	// 获取 Pod
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("无法获取 Pod：%w", err)
	}

	// 获取 Events
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", name),
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取 Event：%w", err)
	}

	// 获取 Metrics
	var usage *PodUsageInfo
	if utils.HasMetricsServer() {
		if metricsClient := utils.GetMetricsClient(); metricsClient != nil {
			metric, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil && len(metric.Containers) > 0 {
				cpu := formatCPU(metric.Containers[0].Usage.Cpu())
				mem := formatMemory(metric.Containers[0].Usage.Memory())
				usage = &PodUsageInfo{CPU: cpu, Memory: mem}
			}
		}
	}

	// 查找 Service
	service, _ := findServiceForPod(ctx, pod)

	// ✅ 获取日志（默认取最后 100 行）
	logs, err := operatorpod.GetPodLogs(ctx, namespace, name, "", 100)
	if err != nil {
		logs = fmt.Sprintf("⚠️ 无法获取日志：%v", err)
	}

	// 返回结构
	return &PodDescribeInfo{
		Pod:     pod,
		Events:  events.Items,
		Usage:   usage,
		Service: service,
		Logs:    logs, // ✅ 加入日志内容
	}, nil
}
