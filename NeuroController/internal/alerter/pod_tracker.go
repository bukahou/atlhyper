// =======================================================================================
// 📄 alerter/deployment_tracker.go
//
// 🩺 Description:
//     监控 Deployment 下的 Pod 异常状态，并基于持续时间判断是否触发告警。
//     核心逻辑包括：异常记录缓存、告警阈值判断、状态快照导出、异常类型判定等。
//
// ⚙️ Features:
//     - 支持 Deployment 粒度的健康状态追踪
//     - 判断异常 Pod 数是否达到副本数，且异常持续时间超过阈值才触发告警
//     - 提供调试日志和状态快照方法
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

	"go.uber.org/zap"
)

// 🧠 全局 Deployment 状态缓存 + 并发锁
var (
	deploymentStates = make(map[string]*types.DeploymentHealthState) // key 格式为 ns/name
	deployMu         sync.Mutex                                      // 保证线程安全
)

// ✅ 更新 Pod 异常状态，并判断是否满足触发告警的条件
//
// 参数：
//   - namespace: Pod 所属命名空间
//   - podName: Pod 名称
//   - deploymentName: Pod 所属 Deployment
//   - reasonCode: 事件原因（如 NotReady, CrashLoopBackOff）
//   - message: 事件详情信息
//   - eventTime: 异常发生时间（K8s 事件时间）
//
// 返回：
//   - shouldAlert: 是否触发告警
//   - reasonText: 告警原因描述（用于邮件等展示）
func UpdatePodEvent(
	namespace string,
	podName string,
	deploymentName string,
	reasonCode string,
	message string,
	eventTime time.Time,
) (shouldAlert bool, reasonText string) {
	threshold := config.GlobalConfig.Diagnosis.UnreadyThresholdDuration
	deployKey := fmt.Sprintf("%s/%s", namespace, deploymentName)

	deployMu.Lock()
	defer deployMu.Unlock()

	// 👀 初始化 Deployment 状态缓存（首次观测）
	state, exists := deploymentStates[deployKey]
	if !exists {
		state = &types.DeploymentHealthState{
			Namespace:     namespace,
			Name:          deploymentName,
			UnreadyPods:   make(map[string]types.PodStatus),
			ExpectedCount: utils.GetExpectedReplicaCount(namespace, deploymentName),
		}
		deploymentStates[deployKey] = state
	}

	// ⚠️ 记录或清除 Pod 异常状态
	if isSevereStatus(reasonCode) {
		// 记录异常 Pod 状态
		state.UnreadyPods[podName] = types.PodStatus{
			PodName:    podName,
			ReasonCode: reasonCode,
			Message:    message,
			Timestamp:  eventTime,
			LastSeen:   time.Now(),
		}
	} else {
		// 检查 Deployment 是否恢复，若恢复则移除该 Pod
		if ok, err := utils.IsDeploymentRecovered(context.TODO(), namespace, deploymentName); err == nil && ok {
			delete(state.UnreadyPods, podName)
		}
	}

	// 🚨 告警判断逻辑：异常 Pod 数是否达到副本数
	if len(state.UnreadyPods) >= state.ExpectedCount {
		fmt.Printf("🚨 [DEBUG] 异常 Pod 数已达期望副本数：%d/%d\n", len(state.UnreadyPods), state.ExpectedCount)

		if state.FirstObserved.IsZero() {
			state.FirstObserved = time.Now()
			fmt.Printf("🕒 [DEBUG] 首次观测异常，记录时间：%v\n", state.FirstObserved)
		} else {
			elapsed := time.Since(state.FirstObserved)
			fmt.Printf("⏳ [DEBUG] 异常已持续：%v（阈值：%v）\n", elapsed, threshold)
		}

		// ✅ 异常持续时间超过阈值，且未触发过告警
		if time.Since(state.FirstObserved) >= threshold && !state.Confirmed {
			state.Confirmed = true
			fmt.Printf("✅ [DEBUG] 满足告警条件，准备发送告警：%s\n", deploymentName)
			return true, fmt.Sprintf("🚨 服务 %s 所有副本异常，已持续 %.0f 秒，请查看完整告警日志", deploymentName, threshold.Seconds())
		} else {
			fmt.Println("🕒 [DEBUG] 尚未满足告警持续时间或已确认过告警，跳过发送")
		}
	} else {
		// 🧹 异常数量回落，清除状态
		fmt.Printf("✅ [DEBUG] 异常 Pod 数未达阈值（%d/%d），清除首次观测时间\n", len(state.UnreadyPods), state.ExpectedCount)
		state.FirstObserved = time.Time{}
		state.Confirmed = false
	}

	// ℹ️ 日志记录：未触发告警
	utils.Info(context.TODO(), "ℹ️ 跳过邮件发送，本次未达到告警条件",
		zap.String("deployment", deploymentName),
		zap.String("namespace", namespace),
		zap.Int("异常Pod数", len(state.UnreadyPods)),
		zap.Int("期望副本数", state.ExpectedCount),
	)

	return false, ""
}

// ✅ 判断是否为严重异常状态（可扩展支持更多 Reason）
func isSevereStatus(reasonCode string) bool {
	switch reasonCode {
	case "NotReady", "CrashLoopBackOff", "ImagePullBackOff", "Failed":
		return true
	default:
		return false
	}
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
