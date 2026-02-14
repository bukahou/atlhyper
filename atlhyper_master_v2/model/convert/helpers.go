// atlhyper_master_v2/model/convert/helpers.go
// convert 包共用辅助函数
package convert

import (
	"fmt"
	"time"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

// formatTimePtr 格式化 *time.Time 指针，nil 返回空字符串
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(timeFormat)
}

// formatAge 根据 CreatedAt 计算相对时间（如 "1d", "2h", "45m"）
func formatAge(created time.Time) string {
	if created.IsZero() {
		return ""
	}
	d := time.Since(created)
	if d < time.Minute {
		return "0m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
