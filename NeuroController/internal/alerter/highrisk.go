// =======================================================================================
// 📄 alerter/deployment_tracker.go
//
// 🩺 Description:
//     Monitors abnormal Pod statuses under a Deployment and determines whether to trigger
//     an alert based on the duration of the issue. Core logic includes caching abnormal
//     states, threshold evaluation, state snapshots, and severity classification.
//
// ⚙️ Features:
//     - Tracks health status at the Deployment level
//     - Triggers alerts only when abnormal Pod count meets replica threshold *and*
//       the issue persists beyond a configured duration
//     - Provides debug logs and snapshot export functions
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package alerter

import (
	"NeuroController/config"
	"NeuroController/internal/types"
	"NeuroController/internal/utils"
	"context"
	"fmt"
	"sync"
	"time"
)

// 🧠 全局 Deployment 状态缓存 + 并发锁
// 用于记录每个 Deployment 的异常 Pod 状态，避免重复告警
var (
	deploymentStates = make(map[string]*types.DeploymentHealthState) // key: namespace/deploymentName
	deployMu         sync.Mutex                                      // 保证并发安全
)

// ✅ 判断是否为严重异常状态（可扩展支持更多 Reason）
// 当前仅处理以下类型的事件作为严重异常
func isSevereStatus(reasonCode string) bool {
	switch reasonCode {
	case "NotReady", "CrashLoopBackOff", "ImagePullBackOff", "Failed":
		return true
	default:
		return false
	}
}

// ✅ 更新 Pod 异常状态，并判断是否满足触发告警的条件
//
// 功能：
//   - 维护当前 Deployment 的异常 Pod 列表（UnreadyPods）
//   - 判断是否“所有副本都异常” 且 “持续时间超过阈值”
//   - 避免短暂波动或局部异常误触发告警
//
// 参数：
//   - namespace: Pod 所属命名空间
//   - podName: 当前异常 Pod 的名称
//   - deploymentName: Pod 所属的 Deployment 名称
//   - reasonCode: K8s 事件的 reason，如 CrashLoopBackOff、NotReady
//   - message: 事件附带的详细信息（可用于告警文案）
//   - eventTime: 事件在 K8s 中发生的时间（用于记录异常起始）
//
// 返回值：
//   - shouldAlert: 是否触发告警
//   - reasonText: 告警原因简要描述（用于组装告警文案）
func UpdatePodEvent(namespace string, podName string, deploymentName string, reasonCode string, message string, eventTime time.Time) (bool, string) {
	ctx := context.TODO()
	threshold := config.GlobalConfig.Diagnosis.UnreadyThresholdDuration // 告警触发的持续时间阈值
	ratioThreshold := config.GlobalConfig.Diagnosis.UnreadyReplicaPercent
	deployKey := fmt.Sprintf("%s/%s", namespace, deploymentName) // 构建唯一 Deployment 键

	deployMu.Lock()
	defer deployMu.Unlock()

	// 🧠 初始化 Deployment 状态缓存
	state, exists := deploymentStates[deployKey]
	if !exists {
		state = &types.DeploymentHealthState{
			Namespace:     namespace,
			Name:          deploymentName,
			UnreadyPods:   make(map[string]types.PodStatus),
			ExpectedCount: utils.GetExpectedReplicaCount(namespace, deploymentName), // 从 K8s API 获取副本数
		}
		deploymentStates[deployKey] = state
	}

	// ⚠️ 如果是严重异常（如 NotReady、CrashLoopBackOff 等），记录异常 Pod 状态
	if isSevereStatus(reasonCode) {
		state.UnreadyPods[podName] = types.PodStatus{
			PodName:    podName,
			ReasonCode: reasonCode,
			Message:    message,
			Timestamp:  eventTime,  // K8s 原始时间
			LastSeen:   time.Now(), // 记录当前观测到的时间
		}
	} else {
		// ✅ 如果当前 Pod 状态不再异常，检查是否整个 Deployment 已恢复
		if ok, err := utils.IsDeploymentRecovered(ctx, namespace, deploymentName); err == nil && ok {
			// 🌱 恢复后从缓存中移除该异常 Pod
			delete(state.UnreadyPods, podName)
		}
	}

	// ✅ 异常副本数达到配置的告警比例阈值时，进入告警判断逻辑
	unready := len(state.UnreadyPods)
	expected := state.ExpectedCount

	if expected > 0 && float64(unready)/float64(expected) >= ratioThreshold {
		// 初次观测异常时记录时间
		if state.FirstObserved.IsZero() {
			state.FirstObserved = time.Now()
		}

		// 若异常持续时间超过阈值且未发送过告警，则触发告警
		if time.Since(state.FirstObserved) >= threshold && !state.Confirmed {
			state.Confirmed = true // 标记已告警，避免重复发送
			return true, fmt.Sprintf("🚨 サービス %s の異常レプリカ率が %.0f%% に達し、%.0f 秒以上継続しています。詳細なアラートログをご確認ください。",
				deploymentName, ratioThreshold*100, threshold.Seconds())
		}
	} else {
		// 异常未达到比例阈值或已恢复，重置异常起始时间与告警标志
		state.FirstObserved = time.Time{}
		state.Confirmed = false
	}

	// 默认不触发告警
	return false, ""
}

// =======================================================================================
// ✅ GetDeploymentStatesSnapshot
//
// 📌 函数功能：
//   - 返回当前所有 Deployment 的健康状态快照（map 格式，key 为 namespace/name）。
//   - 用于对外暴露观察视图，不影响内部原始状态。
//   - 生成的快照是结构体的“深拷贝”，防止外部调用者无意修改内部状态（防御性设计）。
//
// 🧭 使用场景建议（虽然当前尚未使用）：
//   - 🖥️ 提供 REST API 接口供前端查看 Deployment 告警状态。
//   - 🧪 单元测试中对告警状态的断言与验证。
//   - 🧰 调试或诊断工具用于导出当前状态。
//   - 📊 未来用于 Grafana 或可视化界面定期拉取告警状态。
//
// 🔒 并发安全：函数内通过 deployMu 锁保护状态一致性。
//
// 🧠 为何需要深拷贝？
//   - 原始 deploymentStates 中的结构是长期持久的状态缓存（控制器内部使用）
//   - 外部调用者若误修改 map 或 slice 指针会造成状态紊乱，因此返回副本是一种标准的保护机制
//
// =======================================================================================
func GetDeploymentStatesSnapshot() map[string]types.DeploymentHealthState {
	deployMu.Lock()
	defer deployMu.Unlock()

	snapshot := make(map[string]types.DeploymentHealthState)

	for key, val := range deploymentStates {
		// 🔁 深拷贝 UnreadyPods map，防止调用方篡改状态
		clonedPods := make(map[string]types.PodStatus)
		for pod, status := range val.UnreadyPods {
			clonedPods[pod] = status
		}

		// ✅ 构造只读快照副本
		snapshot[key] = types.DeploymentHealthState{
			Namespace:     val.Namespace,
			Name:          val.Name,
			ExpectedCount: val.ExpectedCount,
			UnreadyPods:   clonedPods,
			FirstObserved: val.FirstObserved,
			Confirmed:     val.Confirmed,
		}
	}

	return snapshot
}

// ✅ 判断是否为严重异常状态（可扩展支持更多 Reason）
// func isSevereStatus(reasonCode string) bool {
// 	switch reasonCode {
// 	case "NotReady", "CrashLoopBackOff", "ImagePullBackOff", "Failed":
// 		return true
// 	default:
// 		return false
// 	}
// }
