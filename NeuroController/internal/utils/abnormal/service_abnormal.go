// =======================================================================================
// 📄 abnormal/service_abnormal.go
//
// ✨ 功能说明：
//     定义 Service 相关的异常检测规则及结构体。
//     可供 controller、diagnosis 等模块共用，避免重复定义。
// =======================================================================================

package abnormal

// ✅ 异常结构体：ServiceAbnormalCheck
//
// 描述一条 Service 的异常规则，包含名称、判断逻辑、严重性和提示信息。

type ServiceAbnormalReason struct {
	Code     string
	Message  string
	Severity string
}

var ServiceAbnormalReasonMap = map[string]ServiceAbnormalReason{
	"EmptySelector": {
		Code:     "EmptySelector",
		Message:  "Service 未关联任何 Pod（Selector 为空）",
		Severity: "warning",
	},
	"ClusterIPNone": {
		Code:     "ClusterIPNone",
		Message:  "Service ClusterIP 异常（为空或 None）",
		Severity: "warning",
	},
	"ExternalNameService": {
		Code:     "ExternalNameService",
		Message:  "检测到 ExternalName 类型 Service，可能指向外部服务",
		Severity: "info",
	},
	"PortNotDefined": {
		Code:     "PortNotDefined",
		Message:  "Service 未定义任何端口",
		Severity: "warning",
	},
	"SelectorMismatch": {
		Code:     "SelectorMismatch",
		Message:  "Service Selector 定义但无匹配 Pod，可能配置错误",
		Severity: "warning",
	},
}
