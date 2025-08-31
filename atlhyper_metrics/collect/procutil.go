package collect

import "os"

// ✅ 公共变量：所有指标模块共享
var procRoot = getProcRoot()

// ✅ 允许通过环境变量 PROC_ROOT 覆盖宿主机 /proc 路径
func getProcRoot() string {
	if root := os.Getenv("PROC_ROOT"); root != "" {
		return root
	}
	return "/proc"
}
