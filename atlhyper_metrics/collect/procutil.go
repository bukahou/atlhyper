package collect

import "AtlHyper/atlhyper_metrics/config"

// ProcRoot 返回 /proc 路径（从配置获取）
func ProcRoot() string {
	return config.C.Collect.ProcRoot
}
