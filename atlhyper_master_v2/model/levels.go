// levels.go 读数 → 等级的公共规则（后端判定，前端只按等级上色）
package model

// LevelByThreshold 读数 → 等级：≥ crit 为 crit，≥ warn 为 warn，其余 good。
// 各类读数的阈值不同，分档规则只有这一份
func LevelByThreshold(v, warn, crit float64) HardwareStatus {
	switch {
	case v >= crit:
		return HardwareCrit
	case v >= warn:
		return HardwareWarn
	default:
		return HardwareGood
	}
}

// WarnWhen / CritWhen 单线判定：条件成立即为该等级，否则 good
func WarnWhen(cond bool) HardwareStatus {
	if cond {
		return HardwareWarn
	}
	return HardwareGood
}

func CritWhen(cond bool) HardwareStatus {
	if cond {
		return HardwareCrit
	}
	return HardwareGood
}

// 使用率类读数（CPU / 内存 / swap / 磁盘容量与 inode / 磁盘繁忙度 / 文件句柄 / 连接跟踪表）的告警线：
// 全项目只有这一份，硬件矩阵、对比表、节点详情、概览的节点用量都用它
const (
	ResourceWarnPct = 80.0
	ResourceCritPct = 90.0
)

// ResourceLevel 使用率 → 等级
func ResourceLevel(pct float64) HardwareStatus {
	return LevelByThreshold(pct, ResourceWarnPct, ResourceCritPct)
}
