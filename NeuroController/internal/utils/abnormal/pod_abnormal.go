// 📁 internal/utils/abnormal/pod_abnormal.go

package abnormal

// PodAbnormalReason 表示一个异常原因的详细信息
type PodAbnormalReason struct {
	Code     string // 原始原因字符串（K8s中的 Reason）
	Category string // 所属类别，例如 Waiting / Terminated
	Severity string // 严重等级：critical / warning / info
	Message  string // 可选的用户友好描述
}

var PodAbnormalReasons = map[string]PodAbnormalReason{
	// === Waiting 状态 ===
	"CrashLoopBackOff": {
		Code:     "CrashLoopBackOff",
		Category: "Waiting",
		Severity: "critical",
		Message:  "容器反复崩溃重启",
	},
	"ImagePullBackOff": {
		Code:     "ImagePullBackOff",
		Category: "Waiting",
		Severity: "warning",
		Message:  "镜像拉取失败，进入退避状态",
	},
	"ErrImagePull": {
		Code:     "ErrImagePull",
		Category: "Waiting",
		Severity: "warning",
		Message:  "镜像拉取失败",
	},
	"CreateContainerError": {
		Code:     "CreateContainerError",
		Category: "Waiting",
		Severity: "critical",
		Message:  "容器创建失败",
	},

	// === Terminated 状态 ===
	"OOMKilled": {
		Code:     "OOMKilled",
		Category: "Terminated",
		Severity: "critical",
		Message:  "容器因内存溢出被杀死",
	},
	"Error": {
		Code:     "Error",
		Category: "Terminated",
		Severity: "warning",
		Message:  "容器异常终止退出",
	},
}
