// =======================================================================================
// 📄 abnormal_utils.go
//
// ✨ 功能说明：
//     通用异常辅助函数（目前支持 Pod 异常主因提取）
// =======================================================================================

package abnormal

import (
	"NeuroController/internal/utils"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ✅ 提取 Pod 中首个识别的主要异常原因（返回结构体）
func GetPodAbnormalReason(pod corev1.Pod) *PodAbnormalReason {
	for _, cs := range pod.Status.ContainerStatuses {
		// === 检查 Waiting 状态 ===
		if cs.State.Waiting != nil {
			reason, ok := PodAbnormalReasons[cs.State.Waiting.Reason]
			if ok {
				// 冷却时间判断
				exceptionID := utils.GenerateExceptionID("Pod", pod.Name, pod.Namespace, reason.Code)
				if !utils.ShouldProcessException(exceptionID, time.Now(), 2*time.Minute) {
					return nil
				}
				return &reason
			}
		}

		// === 检查 Terminated 状态 ===
		if cs.State.Terminated != nil {
			reason, ok := PodAbnormalReasons[cs.State.Terminated.Reason]
			if ok {
				exceptionID := utils.GenerateExceptionID("Pod", pod.Name, pod.Namespace, reason.Code)
				if !utils.ShouldProcessException(exceptionID, time.Now(), 2*time.Minute) {
					return nil
				}
				return &reason
			}
		}
	}
	return nil
}

// ✅ 提取 Node 中首个识别的主要异常原因（返回结构体）
func GetNodeAbnormalReason(node corev1.Node) *NodeAbnormalReason {
	for _, cond := range node.Status.Conditions {
		reason, ok := NodeAbnormalConditions[cond.Type]
		if !ok {
			continue
		}

		// 判断是否满足异常条件
		if reason.Category == "Fatal" &&
			(cond.Status == corev1.ConditionFalse || cond.Status == corev1.ConditionUnknown) {

			// 去重判断（如在冷却期内就 return nil）
			exceptionID := utils.GenerateExceptionID("Node", node.Name, "", reason.Code)
			if !utils.ShouldProcessException(exceptionID, time.Now(), 2*time.Minute) {
				return nil
			}
			return &reason
		}

		if reason.Category == "Warning" && cond.Status == corev1.ConditionTrue {
			exceptionID := utils.GenerateExceptionID("Node", node.Name, "", reason.Code)
			if !utils.ShouldProcessException(exceptionID, time.Now(), 2*time.Minute) {
				return nil
			}
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

	// ✅ 生成异常唯一指纹
	exceptionID := utils.GenerateExceptionID("Event", event.InvolvedObject.Name, event.InvolvedObject.Namespace, reason.Code)

	// ✅ 冷却窗口判断（默认 2 分钟）
	if !utils.ShouldProcessException(exceptionID, time.Now(), 2*time.Minute) {
		return nil
	}

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
		exceptionID := utils.GenerateExceptionID("Deployment", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}
		return &reason
	}

	// === 异常 2：Ready 副本不足（实际 Ready < 期望） ===
	if deploy.Status.ReadyReplicas < *deploy.Spec.Replicas {
		reason := DeploymentAbnormalReasons["ReadyReplicaMismatch"]
		exceptionID := utils.GenerateExceptionID("Deployment", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}
		return &reason
	}

	// === 异常 3：更新超时（ProgressDeadlineExceeded）===
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentProgressing && cond.Reason == "ProgressDeadlineExceeded" {
			reason := DeploymentAbnormalReasons["ProgressDeadlineExceeded"]
			exceptionID := utils.GenerateExceptionID("Deployment", name, namespace, reason.Code)
			if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
				return nil
			}
			return &reason
		}
	}

	// === 异常 4：副本数异常上溢（Replicas > 期望值的 1.5 倍）===
	expected := *deploy.Spec.Replicas
	actual := deploy.Status.Replicas
	if actual > int32(float32(expected)*1.5) {
		reason := DeploymentAbnormalReasons["ReplicaOverflow"]
		exceptionID := utils.GenerateExceptionID("Deployment", name, namespace, reason.Code)
		if !utils.ShouldProcessException(exceptionID, now, 2*time.Minute) {
			return nil
		}
		return &reason
	}

	return nil
}
