// atlhyper_master_v2/query/metrics_format.go
// Metrics 单位格式化工具
package query

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatCPU 格式化 CPU 使用量
//
// 输入格式: "123456789n" (纳核) 或 "100m" (毫核)
// 输出格式: "123m" (毫核) 或 "1.5" (核)
func FormatCPU(raw string) string {
	if raw == "" {
		return ""
	}

	raw = strings.TrimSpace(raw)

	// 处理纳核 (n)
	if strings.HasSuffix(raw, "n") {
		nanoStr := strings.TrimSuffix(raw, "n")
		nano, err := strconv.ParseInt(nanoStr, 10, 64)
		if err != nil {
			return raw
		}
		// 转换为毫核
		milli := nano / 1_000_000
		if milli >= 1000 {
			// 超过 1000m，显示为核
			cores := float64(milli) / 1000
			return fmt.Sprintf("%.2f", cores)
		}
		return fmt.Sprintf("%dm", milli)
	}

	// 处理毫核 (m)
	if strings.HasSuffix(raw, "m") {
		milliStr := strings.TrimSuffix(raw, "m")
		milli, err := strconv.ParseInt(milliStr, 10, 64)
		if err != nil {
			return raw
		}
		if milli >= 1000 {
			cores := float64(milli) / 1000
			return fmt.Sprintf("%.2f", cores)
		}
		return fmt.Sprintf("%dm", milli)
	}

	// 可能已经是核数
	return raw
}

// FormatMemory 格式化内存使用量
//
// 输入格式: "123456Ki" (KiB) 或 "128Mi" (MiB)
// 输出格式: "128Mi" 或 "1.5Gi"
func FormatMemory(raw string) string {
	if raw == "" {
		return ""
	}

	raw = strings.TrimSpace(raw)

	// 处理 Ki (KiB)
	if strings.HasSuffix(raw, "Ki") {
		kiStr := strings.TrimSuffix(raw, "Ki")
		ki, err := strconv.ParseInt(kiStr, 10, 64)
		if err != nil {
			return raw
		}
		// 转换为 Mi
		mi := ki / 1024
		if mi >= 1024 {
			// 超过 1024Mi，显示为 Gi
			gi := float64(mi) / 1024
			return fmt.Sprintf("%.2fGi", gi)
		}
		return fmt.Sprintf("%dMi", mi)
	}

	// 处理 Mi (MiB)
	if strings.HasSuffix(raw, "Mi") {
		miStr := strings.TrimSuffix(raw, "Mi")
		mi, err := strconv.ParseInt(miStr, 10, 64)
		if err != nil {
			return raw
		}
		if mi >= 1024 {
			gi := float64(mi) / 1024
			return fmt.Sprintf("%.2fGi", gi)
		}
		return fmt.Sprintf("%dMi", mi)
	}

	// 处理 Gi (GiB)
	if strings.HasSuffix(raw, "Gi") {
		return raw
	}

	// 可能是纯数字（字节）
	bytes, err := strconv.ParseInt(raw, 10, 64)
	if err == nil {
		mi := bytes / (1024 * 1024)
		if mi >= 1024 {
			gi := float64(mi) / 1024
			return fmt.Sprintf("%.2fGi", gi)
		}
		if mi > 0 {
			return fmt.Sprintf("%dMi", mi)
		}
		ki := bytes / 1024
		return fmt.Sprintf("%dKi", ki)
	}

	return raw
}
