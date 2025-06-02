// =======================================================================================
// 📄 event_abnormal.go
//
// ✨ 功能说明：
//     定义 Kubernetes 中常见的 Warning 级别 Event.Reason，用于异常识别与去重。
// =======================================================================================

package abnormal

// ✅ 被视为异常的 Warning 类型事件 Reason 列表
var EventAbnormalReasons = map[string]bool{
	"FailedScheduling":       true,
	"BackOff":                true,
	"ErrImagePull":           true,
	"ImagePullBackOff":       true,
	"FailedCreatePodSandBox": true,
	"FailedMount":            true,
	"FailedAttachVolume":     true,
	"FailedMapVolume":        true,
	"Unhealthy":              true,
	"FailedKillPod":          true,
	"Failed":                 true,
}
