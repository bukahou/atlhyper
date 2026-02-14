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

// formatDuration 计算两个时间之间的持续时间（如 "5m32s", "2h30m"）
func formatDuration(start, end *time.Time) string {
	if start == nil || end == nil {
		return ""
	}
	d := end.Sub(*start)
	if d < 0 {
		return ""
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// formatTimeAgo 计算距今的相对时间（如 "10m", "3h", "2d"），复用 formatAge 逻辑
func formatTimeAgo(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatAge(*t)
}
