package abnormal

// ✅ Deployment 异常结构体
type DeploymentAbnormalReason struct {
	Code     string // 异常代码（如 UnavailableReplica）
	Message  string // 可读性更强的提示文本
	Category string // 异常分类（Fatal / Warning / Info）
	Severity string // 异常等级（critical / warning / info）
}

// ✅ Deployment 异常识别表（可按字段触发映射）
var DeploymentAbnormalReasons = map[string]DeploymentAbnormalReason{
	"UnavailableReplica": {
		Code:     "UnavailableReplica",
		Message:  "Deployment 存在不可用副本，可能为镜像拉取失败、Pod 崩溃等",
		Category: "Warning",
		Severity: "info",
	},
	"ReadyReplicaMismatch": {
		Code:     "ReadyReplicaMismatch",
		Message:  "Ready 副本数不足，未达到期望副本数",
		Category: "Warning",
		Severity: "warning",
	},
	"ProgressDeadlineExceeded": {
		Code:     "ProgressDeadlineExceeded",
		Message:  "Deployment 更新超时，未在期望时间内完成滚动更新",
		Category: "Fatal",
		Severity: "critical",
	},
	"ReplicaOverflow": {
		Code:     "ReplicaOverflow",
		Message:  "Deployment 实际副本数远超期望，可能为滚动更新异常或旧副本未缩容",
		Category: "Warning",
		Severity: "warning",
	},
}
