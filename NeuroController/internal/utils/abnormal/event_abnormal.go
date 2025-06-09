// =======================================================================================
// 📄 event_abnormal.go
//
// ✨ 功能说明：
//     定义 Kubernetes 中常见的 Warning 级别 Event.Reason，用于异常识别与去重。
// =======================================================================================

package abnormal

import (
	"sync"
	"time"
)

// EventAbnormalReason 表示 Kubernetes Warning 事件的详细结构
type EventAbnormalReason struct {
	Code     string // 原始 Reason（如 "FailedScheduling"）
	Severity string // 严重等级：critical / warning / info
	Message  string // 用户友好的描述
}

// EventAbnormalReasons 映射表：已识别的异常事件
var EventAbnormalReasons = map[string]EventAbnormalReason{
	"FailedScheduling": {
		Code:     "FailedScheduling",
		Severity: "warning",
		Message:  "Pod 调度失败，可能资源不足或亲和性规则不匹配",
	},
	"BackOff": {
		Code:     "BackOff",
		Severity: "critical",
		Message:  "容器多次启动失败，进入退避重试状态",
	},
	"ErrImagePull": {
		Code:     "ErrImagePull",
		Severity: "warning",
		Message:  "镜像拉取失败，可能是镜像不存在或网络异常",
	},
	"ImagePullBackOff": {
		Code:     "ImagePullBackOff",
		Severity: "warning",
		Message:  "镜像拉取失败并进入退避状态",
	},
	"FailedCreatePodSandBox": {
		Code:     "FailedCreatePodSandBox",
		Severity: "critical",
		Message:  "Pod 沙箱创建失败，可能是容器运行时或 CNI 插件异常",
	},
	"FailedMount": {
		Code:     "FailedMount",
		Severity: "warning",
		Message:  "卷挂载失败，可能路径不存在或权限不足",
	},
	"FailedAttachVolume": {
		Code:     "FailedAttachVolume",
		Severity: "warning",
		Message:  "卷附加失败，常见于 PVC / PV / 云盘等场景",
	},
	"FailedMapVolume": {
		Code:     "FailedMapVolume",
		Severity: "warning",
		Message:  "卷映射失败，可能挂载点配置有误",
	},
	"Unhealthy": {
		Code:     "Unhealthy",
		Severity: "critical",
		Message:  "健康检查未通过，容器状态异常",
	},
	"FailedKillPod": {
		Code:     "FailedKillPod",
		Severity: "warning",
		Message:  "无法终止 Pod，可能是进程卡死或 runtime 异常",
	},
	"Failed": {
		Code:     "Failed",
		Severity: "warning",
		Message:  "操作失败（通用原因）",
	},
}

// ShouldTriggerUnhealthyWithinWindow：在 timeWindow 内连续触发 threshold 次才允许告警
func ShouldTriggerUnhealthyWithinWindow(id string, threshold int, timeWindow time.Duration) bool {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()

	// 冷却期间不重复告警
	if last, ok := lastUnhealthyFired[id]; ok && now.Sub(last) < cooldown {
		return false
	}

	// 获取时间列表并追加本次
	times := unhealthyTimestamps[id]
	times = append(times, now)

	// 保留 timeWindow 内的时间戳
	filtered := make([]time.Time, 0, len(times))
	for _, t := range times {
		if now.Sub(t) <= timeWindow {
			filtered = append(filtered, t)
		}
	}
	unhealthyTimestamps[id] = filtered

	// 判断是否达到阈值
	if len(filtered) >= threshold {
		unhealthyTimestamps[id] = []time.Time{} // 触发后清空计数
		lastUnhealthyFired[id] = now
		return true
	}

	return false
}

var (
	unhealthyTimestamps = make(map[string][]time.Time)
	lastUnhealthyFired  = make(map[string]time.Time)
	mu                  sync.Mutex
	cooldown            = 5 * time.Minute
)
