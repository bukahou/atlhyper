package alerter

import (
	"NeuroController/config"
	"NeuroController/internal/utils"
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ⚙️ 触发告警所需的最小持续时间（异常持续多久才算告警）
// const unreadyThresholdDuration = 30 * time.Second

// 🧠 全局状态缓存 + 并发锁
var (
	deploymentStates = make(map[string]*DeploymentHealthState) // key: ns/name
	deployMu         sync.Mutex
)

// ✅ 更新 Pod 异常状态，并判断是否触发告警
func UpdatePodEvent(
	namespace string,
	podName string,
	deploymentName string,
	reasonCode string, // 如 "Unhealthy", "ReadinessProbeFailed"
	message string, // 原始异常信息
	eventTime time.Time,
) (shouldAlert bool, reasonText string) {

	threshold := config.GlobalConfig.Diagnosis.UnreadyThresholdDuration

	deployKey := fmt.Sprintf("%s/%s", namespace, deploymentName)

	deployMu.Lock()
	defer deployMu.Unlock()

	state, exists := deploymentStates[deployKey]
	if !exists {
		state = &DeploymentHealthState{
			Namespace:     namespace,
			Name:          deploymentName,
			UnreadyPods:   make(map[string]PodStatus),
			ExpectedCount: utils.GetExpectedReplicaCount(namespace, deploymentName),
		}
		deploymentStates[deployKey] = state
	}

	// 更新状态
	if isSevereStatus(reasonCode) {
		state.UnreadyPods[podName] = PodStatus{
			PodName:    podName,
			reasonCode: reasonCode,
			Message:    message,
			Timestamp:  eventTime,
			LastSeen:   time.Now(),
		}
	} else {
		if ok, err := utils.IsDeploymentRecovered(context.TODO(), namespace, deploymentName); err == nil && ok {
			delete(state.UnreadyPods, podName)
		}

	}

	// 判断是否触发告警
	if len(state.UnreadyPods) >= state.ExpectedCount {
		fmt.Printf("🚨 [DEBUG] 异常 Pod 数已达期望副本数：%d/%d\n", len(state.UnreadyPods), state.ExpectedCount)

		if state.FirstObserved.IsZero() {
			state.FirstObserved = time.Now()
			fmt.Printf("🕒 [DEBUG] 首次观测异常，记录时间：%v\n", state.FirstObserved)
		} else {
			elapsed := time.Since(state.FirstObserved)
			fmt.Printf("⏳ [DEBUG] 异常已持续：%v（阈值：%v）\n", elapsed, threshold)
		}

		if time.Since(state.FirstObserved) >= threshold && !state.Confirmed {
			state.Confirmed = true
			fmt.Printf("✅ [DEBUG] 满足告警条件，准备发送告警：%s\n", deploymentName)
			return true, fmt.Sprintf("🚨 服务 %s 所有副本异常，已持续 %.0f 秒，请查看完整告警日志", deploymentName, threshold.Seconds())
		} else {
			fmt.Println("🕒 [DEBUG] 尚未满足告警持续时间或已确认过告警，跳过发送")
		}

	} else if len(state.UnreadyPods) < state.ExpectedCount {
		fmt.Printf("✅ [DEBUG] 异常 Pod 数未达阈值（%d/%d），清除首次观测时间\n", len(state.UnreadyPods), state.ExpectedCount)
		state.FirstObserved = time.Time{}
		state.Confirmed = false
	}

	// ✅ 加入未触发告警的调试日志
	utils.Info(context.TODO(), "ℹ️ 跳过邮件发送，本次未达到告警条件",
		zap.String("deployment", deploymentName),
		zap.String("namespace", namespace),
		zap.Int("异常Pod数", len(state.UnreadyPods)),
		zap.Int("期望副本数", state.ExpectedCount),
	)

	return false, ""

}

// ✅ 判断是否属于严重异常状态（可按需扩展）
func isSevereStatus(reasonCode string) bool {
	switch reasonCode {
	case "NotReady", "CrashLoopBackOff", "ImagePullBackOff", "Failed":
		return true
	default:
		return false
	}
}

// ✅ 模拟副本数获取（可接入 Kubernetes API 真实值）
// func GuessExpectedReplicas(deploymentName string) int {
// 	// TODO: 可以替换为实际 Kubernetes 查询
// 	// 临时默认所有 Deployment 都有 2 个副本
// 	return 2
// }

// ✅ 可选：导出状态快照用于诊断展示
func GetDeploymentStatesSnapshot() map[string]DeploymentHealthState {
	deployMu.Lock()
	defer deployMu.Unlock()

	snapshot := make(map[string]DeploymentHealthState)
	for key, val := range deploymentStates {
		// 避免外部修改，复制结构体（map 的深拷贝）
		clonedPods := make(map[string]PodStatus)
		for pod, status := range val.UnreadyPods {
			clonedPods[pod] = status
		}
		snapshot[key] = DeploymentHealthState{
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
