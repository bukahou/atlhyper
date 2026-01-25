// source/abnormal/pod_abnormal.go
// Pod 异常检测
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

	// === 建议新增的 Terminated 状态 ===
	"StartError": {
		Code:     "StartError",
		Category: "Terminated",
		Severity: "critical",
		Message:  "容器启动失败，通常是因执行权限或路径错误",
	},
	"Completed": {
		Code:     "Completed",
		Category: "Terminated",
		Severity: "info",
		Message:  "容器已正常退出（用于 Job）",
	},

	// === 建议新增的 Waiting 状态 ===
	"RunContainerError": {
		Code:     "RunContainerError",
		Category: "Waiting",
		Severity: "critical",
		Message:  "容器运行时出错，可能是镜像或执行文件异常",
	},
	"ContainerCannotRun": {
		Code:     "ContainerCannotRun",
		Category: "Waiting",
		Severity: "critical",
		Message:  "容器启动失败，可能因入口命令错误或缺少可执行文件",
	},
	"InvalidImageName": {
		Code:     "InvalidImageName",
		Category: "Waiting",
		Severity: "warning",
		Message:  "镜像名不合法或格式错误",
	},
	"CreateContainerConfigError": {
		Code:     "CreateContainerConfigError",
		Category: "Waiting",
		Severity: "critical",
		Message:  "容器配置错误导致无法创建",
	},
	"EmptyContainerStatus": {
		Code:     "EmptyContainerStatus",
		Category: "Init",
		Severity: "info",
		Message:  "容器状态尚未建立，可能仍在调度或镜像尚未拉取",
	},
	"ReadinessProbeFailed": {
		Code:     "ReadinessProbeFailed",
		Category: "Condition",
		Severity: "warning",
		Message:  "Readiness 探针检测失败，服务未就绪",
	},
	"NotReady": {
		Code:     "NotReady",
		Category: "Condition",
		Severity: "warning",
		Message:  "Pod 未就绪，可能原因未知或未上报",
	},
}
