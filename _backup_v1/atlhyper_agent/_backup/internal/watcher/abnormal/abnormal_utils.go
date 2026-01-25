package abnormal

import (
	"AtlHyper/atlhyper_agent/utils"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
)

// ✅ 提取 Pod 中首个识别的主要异常原因（返回结构体）
func GetPodAbnormalReason(pod corev1.Pod) *PodAbnormalReason {
	now := time.Now()

	// === 检查 Container 状态 ===
	for _, cs := range pod.Status.ContainerStatuses {
		// === Waiting 状态 ===
		if cs.State.Waiting != nil {
			reasonCode := cs.State.Waiting.Reason
			if reason, ok := PodAbnormalReasons[reasonCode]; ok {
				// fmt.Printf("🧩 [异常识别] Waiting 状态：%s/%s → Reason=%s（候选）\n", pod.Namespace, pod.Name, reasonCode)

				exceptionID := utils.GenerateExceptionID("Pod", pod.Name, pod.Namespace, reason.Code)
				if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
					return nil
				}

				// fmt.Printf(" [异常识别] Waiting 状态：%s/%s → Reason=%s，Message=%s（已确认）\n", pod.Namespace, pod.Name, reasonCode, reason.Message)
				return &reason
			}
		}

		// === Terminated 状态 ===
		if cs.State.Terminated != nil {
			reasonCode := cs.State.Terminated.Reason
			if reason, ok := PodAbnormalReasons[reasonCode]; ok {
				// fmt.Printf("🧩 [异常识别] Terminated 状态：%s/%s → Reason=%s（候选）\n", pod.Namespace, pod.Name, reasonCode)

				exceptionID := utils.GenerateExceptionID("Pod", pod.Name, pod.Namespace, reason.Code)
				if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
					return nil
				}

				// fmt.Printf(" [异常识别] Terminated 状态：%s/%s → Reason=%s，Message=%s（已确认）\n", pod.Namespace, pod.Name, reasonCode, reason.Message)
				return &reason
			}
		}
	}

	// === 检查 Pod Ready=False 状态 ===
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionFalse {
			code := cond.Reason
			reason, ok := PodAbnormalReasons[code]
			if !ok {
				reason = PodAbnormalReasons["NotReady"]
				code = "NotReady"
			}

			// fmt.Printf("🧩 [异常识别] Condition 状态：%s/%s → Reason=%s（候选）\n", pod.Namespace, pod.Name, code)

			exceptionID := utils.GenerateExceptionID("Pod", pod.Name, pod.Namespace, reason.Code)
			if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
				return nil
			}

			// fmt.Printf(" [异常识别] Condition 状态：%s/%s → Reason=%s，Message=%s（已确认）\n", pod.Namespace, pod.Name, code, reason.Message)
			return &reason
		}
	}

	return nil
}

// ✅ 提取 Node 中首个识别的主要异常原因（返回结构体）
func GetNodeAbnormalReason(node corev1.Node) *NodeAbnormalReason {
	now := time.Now()

	for _, cond := range node.Status.Conditions {
		reason, ok := NodeAbnormalConditions[cond.Type]
		if !ok {
			continue
		}

		// === 致命类异常 ===
		if reason.Category == "Fatal" && (cond.Status == corev1.ConditionFalse || cond.Status == corev1.ConditionUnknown) {
			// fmt.Printf("🧩 [异常识别] Node Fatal 状态：%s → Condition=%s（候选）\n", node.Name, cond.Type)

			exceptionID := utils.GenerateExceptionID("Node", node.Name, "", reason.Code)
			if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
				return nil
			}

			// fmt.Printf(" [异常识别] Node Fatal 状态：%s → Condition=%s，Message=%s（已确认）\n", node.Name, cond.Type, reason.Message)
			return &reason
		}

		// === 警告类异常 ===
		if reason.Category == "Warning" && cond.Status == corev1.ConditionTrue {
			// fmt.Printf("🧩 [异常识别] Node Warning 状态：%s → Condition=%s（候选）\n", node.Name, cond.Type)

			exceptionID := utils.GenerateExceptionID("Node", node.Name, "", reason.Code)
			if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
				return nil
			}

			// fmt.Printf(" [异常识别] Node Warning 状态：%s → Condition=%s，Message=%s（已确认）\n", node.Name, cond.Type, reason.Message)
			return &reason
		}
	}

	return nil
}

// =======================================================================================
// ✅ 提取 Event 中已知的异常原因（返回结构体）
// 提取 Kubernetes Event 中的主要异常原因（用于识别 Warning 等）
func GetEventAbnormalReason(event corev1.Event) *EventAbnormalReason {
	reason, ok := EventAbnormalReasons[event.Reason]
	if !ok {
		return nil
	}

	// 新增：只处理最近 N 分钟的事件，避免历史污染
	if time.Since(event.LastTimestamp.Time) > 5*time.Minute {
		return nil
	}

	// fmt.Printf("🧩 [异常识别] Event 状态：%s/%s → Reason=%s（候选）\n", event.InvolvedObject.Namespace, event.InvolvedObject.Name, event.Reason)

	// ✅ 特殊处理 Readiness 异常（提前 return）
	if reason.Code == "Unhealthy" {
		exceptionID := utils.GeneratePodInstanceExceptionID(
			event.InvolvedObject.Namespace,
			event.InvolvedObject.UID,
			reason.Code,
		)
		if !ShouldTriggerUnhealthyWithinWindow(exceptionID, 3, 40*time.Second) {
			return nil
		}
		return &reason
	}

	exceptionID := utils.GenerateExceptionID("Event", event.InvolvedObject.Name, event.InvolvedObject.Namespace, reason.Code)
	if !utils.ShouldProcessException(exceptionID, time.Now(), 2*time.Minute) {
		return nil
	}

	// fmt.Printf(" [异常识别] Event 状态：%s/%s → Reason=%s，Message=%s（已确认）\n", event.InvolvedObject.Namespace, event.InvolvedObject.Name, event.Reason, reason.Message)

	return &reason
}

// ✅ 提取 Deployment 中首个识别的主要异常原因（返回结构体）
func GetDeploymentAbnormalReason(deploy appsv1.Deployment) *DeploymentAbnormalReason {
	now := time.Now()
	name := deploy.Name
	namespace := deploy.Namespace

	// === 异常 1：存在不可用副本 ===
	if deploy.Status.UnavailableReplicas > 0 {
		reason := DeploymentAbnormalReasons["UnavailableReplica"]
		// fmt.Printf("🧩 [异常识别] Deployment 不可用副本：%s/%s → Unavailable=%d（候选）\n", namespace, name, deploy.Status.UnavailableReplicas)

		exceptionID := utils.GenerateExceptionID("Deployment", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}

		// fmt.Printf(" [异常识别] Deployment 不可用副本：%s/%s → Reason=%s，Message=%s（已确认）\n", namespace, name, reason.Code, reason.Message)
		return &reason
	}

	// === 异常 2：Ready 副本不足 ===
	if deploy.Status.ReadyReplicas < *deploy.Spec.Replicas {
		reason := DeploymentAbnormalReasons["ReadyReplicaMismatch"]
		// fmt.Printf("🧩 [异常识别] Deployment Ready 副本不足：%s/%s → Ready=%d / Desired=%d（候选）\n", namespace, name, deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)

		exceptionID := utils.GenerateExceptionID("Deployment", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}

		// fmt.Printf(" [异常识别] Deployment Ready 副本不足：%s/%s → Reason=%s，Message=%s（已确认）\n", namespace, name, reason.Code, reason.Message)
		return &reason
	}

	// === 异常 3：更新超时 ===
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentProgressing && cond.Reason == "ProgressDeadlineExceeded" {
			reason := DeploymentAbnormalReasons["ProgressDeadlineExceeded"]
			// fmt.Printf("🧩 [异常识别] Deployment 更新超时：%s/%s → Reason=%s（候选）\n", namespace, name, cond.Reason)

			exceptionID := utils.GenerateExceptionID("Deployment", name, namespace, reason.Code)
			if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
				return nil
			}

			// fmt.Printf("[异常识别] Deployment 更新超时：%s/%s → Reason=%s，Message=%s（已确认）\n", namespace, name, reason.Code, reason.Message)
			return &reason
		}
	}

	// === 异常 4：副本上溢 ===
	expected := *deploy.Spec.Replicas
	actual := deploy.Status.Replicas
	if actual > int32(float32(expected)*1.5) {
		reason := DeploymentAbnormalReasons["ReplicaOverflow"]
		// fmt.Printf("🧩 [异常识别] Deployment 副本数上溢：%s/%s → Actual=%d > Expected=%.1f（候选）\n", namespace, name, actual, float32(expected)*1.5)

		exceptionID := utils.GenerateExceptionID("Deployment", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}

		// fmt.Printf(" [异常识别] Deployment 副本数上溢：%s/%s → Reason=%s，Message=%s（已确认）\n", namespace, name, reason.Code, reason.Message)
		return &reason
	}

	return nil
}

// func GetEndpointAbnormalReason(ep *corev1.Endpoints) *EndpointAbnormalReason {
// 	now := time.Now()

// 	for _, rule := range EndpointAbnormalRules {
// 		if rule.Check(ep) {
// 			// fmt.Printf("🧩 [异常识别] Endpoints 状态异常：%s/%s → Rule=%s（候选）\n", ep.Namespace, ep.Name, rule.Code)

// 			exceptionID := utils.GenerateExceptionID("Endpoints", ep.Name, ep.Namespace, rule.Code)
// 			if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
// 				return nil
// 			}

// 			// fmt.Printf(" [异常识别] Endpoints 状态异常：%s/%s → Code=%s，Message=%s（已确认）\n", ep.Namespace, ep.Name, rule.Code, rule.Message)

// 			return &EndpointAbnormalReason{
// 				Code:     rule.Code,
// 				Message:  rule.Message,
// 				Severity: rule.Severity,
// 			}
// 		}
// 	}
// 	return nil
// }

// ✅ EndpointSlice 异常判定逻辑（保留原函数名）
func GetEndpointAbnormalReason(slice *discoveryv1.EndpointSlice) *EndpointAbnormalReason {
	now := time.Now()

	for _, rule := range EndpointAbnormalRules {
		if rule.Check(slice) {
			exceptionID := utils.GenerateExceptionID("EndpointSlice", slice.Name, slice.Namespace, rule.Code)
			if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
				return nil
			}

			return &EndpointAbnormalReason{
				Code:     rule.Code,
				Message:  rule.Message,
				Severity: rule.Severity,
			}
		}
	}

	return nil
}

func GetServiceAbnormalReason(svc corev1.Service) *ServiceAbnormalReason {
	now := time.Now()
	name := svc.Name
	namespace := svc.Namespace

	if name == "kubernetes" && namespace == "default" {
		return nil
	}

	//  异常 1：Selector 为空
	if len(svc.Spec.Selector) == 0 {
		reason := ServiceAbnormalReasonMap["EmptySelector"]
		exceptionID := utils.GenerateExceptionID("Service", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}
		return &reason
	}

	//  异常 2：ExternalName 类型
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		reason := ServiceAbnormalReasonMap["ExternalNameService"]
		exceptionID := utils.GenerateExceptionID("Service", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}
		return &reason
	}

	//  异常 3：ClusterIP 异常
	if svc.Spec.ClusterIP == "" || svc.Spec.ClusterIP == "None" {
		reason := ServiceAbnormalReasonMap["ClusterIPNone"]
		exceptionID := utils.GenerateExceptionID("Service", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}
		return &reason
	}

	//  异常 4：未定义任何端口
	if len(svc.Spec.Ports) == 0 {
		reason := ServiceAbnormalReasonMap["PortNotDefined"]
		exceptionID := utils.GenerateExceptionID("Service", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}
		return &reason
	}

	return nil
}
