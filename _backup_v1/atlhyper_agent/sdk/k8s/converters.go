// sdk/k8s/converters.go
// K8s 类型到 SDK 类型的转换
package k8s

import (
	"AtlHyper/atlhyper_agent/sdk"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ==================== Pod 转换 ====================

func convertPod(pod *corev1.Pod) sdk.PodInfo {
	info := sdk.PodInfo{
		Meta:     convertMeta(&pod.ObjectMeta),
		Phase:    string(pod.Status.Phase),
		NodeName: pod.Spec.NodeName,
		PodIP:    pod.Status.PodIP,
		HostIP:   pod.Status.HostIP,
	}

	// 转换容器信息
	for _, c := range pod.Spec.Containers {
		ci := sdk.ContainerInfo{
			Name:  c.Name,
			Image: c.Image,
		}
		// 查找对应的状态
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == c.Name {
				ci.Ready = cs.Ready
				ci.RestartCount = cs.RestartCount
				ci.State, ci.StateReason, ci.StateMessage = getContainerState(cs.State)
				break
			}
		}
		info.Containers = append(info.Containers, ci)
	}

	// 转换 Conditions
	for _, cond := range pod.Status.Conditions {
		info.Conditions = append(info.Conditions, sdk.PodCondition{
			Type:    string(cond.Type),
			Status:  cond.Status == corev1.ConditionTrue,
			Reason:  cond.Reason,
			Message: cond.Message,
		})
	}

	return info
}

func getContainerState(state corev1.ContainerState) (string, string, string) {
	if state.Running != nil {
		return "Running", "", ""
	}
	if state.Waiting != nil {
		return "Waiting", state.Waiting.Reason, state.Waiting.Message
	}
	if state.Terminated != nil {
		return "Terminated", state.Terminated.Reason, state.Terminated.Message
	}
	return "Unknown", "", ""
}

// ==================== Node 转换 ====================

func convertNode(node *corev1.Node) sdk.NodeInfo {
	info := sdk.NodeInfo{
		Meta:          convertMeta(&node.ObjectMeta),
		Unschedulable: node.Spec.Unschedulable,
		Capacity:      convertResourceList(node.Status.Capacity),
		Allocatable:   convertResourceList(node.Status.Allocatable),
		NodeInfo: sdk.NodeSystemInfo{
			KernelVersion:           node.Status.NodeInfo.KernelVersion,
			OSImage:                 node.Status.NodeInfo.OSImage,
			ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
			Architecture:            node.Status.NodeInfo.Architecture,
			OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
		},
	}

	// 转换地址
	for _, addr := range node.Status.Addresses {
		info.Addresses = append(info.Addresses, sdk.NodeAddress{
			Type:    string(addr.Type),
			Address: addr.Address,
		})
	}

	// 转换 Conditions
	for _, cond := range node.Status.Conditions {
		info.Conditions = append(info.Conditions, sdk.NodeCondition{
			Type:    string(cond.Type),
			Status:  cond.Status == corev1.ConditionTrue,
			Reason:  cond.Reason,
			Message: cond.Message,
		})
	}

	return info
}

// ==================== Deployment 转换 ====================

func convertDeployment(deploy *appsv1.Deployment) sdk.DeploymentInfo {
	info := sdk.DeploymentInfo{
		Meta:              convertMeta(&deploy.ObjectMeta),
		Replicas:          *deploy.Spec.Replicas,
		ReadyReplicas:     deploy.Status.ReadyReplicas,
		AvailableReplicas: deploy.Status.AvailableReplicas,
		UpdatedReplicas:   deploy.Status.UpdatedReplicas,
	}

	// 转换容器规格
	for _, c := range deploy.Spec.Template.Spec.Containers {
		info.Containers = append(info.Containers, sdk.ContainerSpec{
			Name:  c.Name,
			Image: c.Image,
		})
	}

	return info
}

// ==================== Service 转换 ====================

func convertService(svc *corev1.Service) sdk.ServiceInfo {
	info := sdk.ServiceInfo{
		Meta:        convertMeta(&svc.ObjectMeta),
		Type:        string(svc.Spec.Type),
		ClusterIP:   svc.Spec.ClusterIP,
		ExternalIPs: svc.Spec.ExternalIPs,
		Selector:    svc.Spec.Selector,
	}

	// 转换端口
	for _, p := range svc.Spec.Ports {
		info.Ports = append(info.Ports, sdk.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: int32(p.TargetPort.IntValue()),
			NodePort:   p.NodePort,
			Protocol:   string(p.Protocol),
		})
	}

	return info
}

// ==================== Namespace 转换 ====================

func convertNamespace(ns *corev1.Namespace) sdk.NamespaceInfo {
	return sdk.NamespaceInfo{
		Meta:   convertMeta(&ns.ObjectMeta),
		Phase:  string(ns.Status.Phase),
		Labels: ns.Labels,
	}
}

// ==================== ConfigMap 转换 ====================

func convertConfigMap(cm *corev1.ConfigMap) sdk.ConfigMapInfo {
	return sdk.ConfigMapInfo{
		Meta: convertMeta(&cm.ObjectMeta),
		Data: cm.Data,
	}
}

// ==================== Ingress 转换 ====================

func convertIngress(ing *networkingv1.Ingress) sdk.IngressInfo {
	info := sdk.IngressInfo{
		Meta: convertMeta(&ing.ObjectMeta),
	}

	if ing.Spec.IngressClassName != nil {
		info.ClassName = *ing.Spec.IngressClassName
	}

	// 转换规则
	for _, rule := range ing.Spec.Rules {
		r := sdk.IngressRule{
			Host: rule.Host,
		}
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				p := sdk.IngressPath{
					Path: path.Path,
				}
				if path.PathType != nil {
					p.PathType = string(*path.PathType)
				}
				if path.Backend.Service != nil {
					p.ServiceName = path.Backend.Service.Name
					if path.Backend.Service.Port.Number != 0 {
						p.ServicePort = path.Backend.Service.Port.Number
					}
				}
				r.Paths = append(r.Paths, p)
			}
		}
		info.Rules = append(info.Rules, r)
	}

	// 转换 TLS
	for _, tls := range ing.Spec.TLS {
		info.TLS = append(info.TLS, sdk.IngressTLS{
			Hosts:      tls.Hosts,
			SecretName: tls.SecretName,
		})
	}

	// 转换默认后端
	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
		info.DefaultBackend = &sdk.IngressBackend{
			ServiceName: ing.Spec.DefaultBackend.Service.Name,
		}
		if ing.Spec.DefaultBackend.Service.Port.Number != 0 {
			info.DefaultBackend.ServicePort = ing.Spec.DefaultBackend.Service.Port.Number
		}
	}

	return info
}

// ==================== 通用转换 ====================

func convertMeta(meta *metav1.ObjectMeta) sdk.ObjectMeta {
	return sdk.ObjectMeta{
		Name:              meta.Name,
		Namespace:         meta.Namespace,
		UID:               string(meta.UID),
		Labels:            meta.Labels,
		Annotations:       meta.Annotations,
		CreationTimestamp: meta.CreationTimestamp.Time,
	}
}

func convertResourceList(rl corev1.ResourceList) sdk.ResourceList {
	result := sdk.ResourceList{}
	if cpu, ok := rl[corev1.ResourceCPU]; ok {
		result.CPU = cpu.String()
	}
	if mem, ok := rl[corev1.ResourceMemory]; ok {
		result.Memory = mem.String()
	}
	if pods, ok := rl[corev1.ResourcePods]; ok {
		result.Pods = pods.String()
	}
	return result
}
