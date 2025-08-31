package abnormal

import corev1 "k8s.io/api/core/v1"

// ✅ Endpoints 异常结构体
type EndpointAbnormalReason struct {
	Code     string // 异常代码标识（如 NoReadyAddress）
	Message  string // 可读性更强的提示文本
	Severity string // 异常等级（critical / warning）
}

// ✅ Endpoints 异常识别规则
//
// 目前只定义了核心异常：所有 Ready 地址为空
// 可拓展更多如 Subsets 为空、NotReady 地址过多等场景
var EndpointAbnormalRules = []struct {
	Code     string
	Check    func(ep *corev1.Endpoints) bool
	Message  string
	Severity string
}{
	{
		Code: "NoReadyAddress",
		Check: func(ep *corev1.Endpoints) bool {
			total := 0
			ready := 0
			for _, s := range ep.Subsets {
				total += len(s.Addresses) + len(s.NotReadyAddresses)
				ready += len(s.Addresses)
			}
			return total > 0 && ready == 0
		},
		Message:  " 所有 Pod 已从 Endpoints 剔除（无可用后端）",
		Severity: "critical",
	},
	{
		Code: "NoSubsets",
		Check: func(ep *corev1.Endpoints) bool {
			return len(ep.Subsets) == 0
		},
		Message:  " Endpoints 无任何子集（Subsets 为空）",
		Severity: "warning",
	},
}
