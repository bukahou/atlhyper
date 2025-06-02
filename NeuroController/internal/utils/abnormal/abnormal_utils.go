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

	corev1 "k8s.io/api/core/v1"
)

// ✅ 提取 Pod 中首个识别的主要异常原因（返回结构体）
func GetPodAbnormalReason(pod corev1.Pod) *PodAbnormalReason {
	for _, cs := range pod.Status.ContainerStatuses {
		// 检查 Waiting 状态
		if cs.State.Waiting != nil {
			if reason, ok := PodAbnormalWaitingReasons[cs.State.Waiting.Reason]; ok {
				return &reason
			}
		}
		// 检查 Terminated 状态
		if cs.State.Terminated != nil {
			if reason, ok := PodAbnormalTerminatedReasons[cs.State.Terminated.Reason]; ok {
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
