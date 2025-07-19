// =======================================================================================
// 📄 node_abnormal.go
//
// ✨ 功能说明：
//     定义 Node 异常类型结构体与识别表，用于统一提取异常描述与分类。
// =======================================================================================

package abnormal

import corev1 "k8s.io/api/core/v1"

// ✅ Node 异常结构体
type NodeAbnormalReason struct {
	Code     string // 原始 Condition Type 名称（如 NotReady）
	Message  string // 可读性更强的提示文本
	Category string // 异常分类（Fatal / Warning）
	Severity string // 异常等级（critical / warning）
}

// ✅ 异常条件表（Ready=False 或 Unknown 视为致命异常）
var NodeAbnormalConditions = map[corev1.NodeConditionType]NodeAbnormalReason{
	corev1.NodeReady: {
		Code:     "NotReady",
		Message:  "节点未就绪，可能通信中断",
		Category: "Fatal",
		Severity: "critical",
	},
	corev1.NodeMemoryPressure: {
		Code:     "MemoryPressure",
		Message:  "节点内存资源不足",
		Category: "Warning",
		Severity: "warning",
	},
	corev1.NodeDiskPressure: {
		Code:     "DiskPressure",
		Message:  "节点磁盘空间不足",
		Category: "Warning",
		Severity: "warning",
	},
	corev1.NodePIDPressure: {
		Code:     "PIDPressure",
		Message:  "节点进程数耗尽",
		Category: "Warning",
		Severity: "warning",
	},
	corev1.NodeNetworkUnavailable: {
		Code:     "NetworkUnavailable",
		Message:  "节点网络不可用",
		Category: "Warning",
		Severity: "warning",
	},
}
