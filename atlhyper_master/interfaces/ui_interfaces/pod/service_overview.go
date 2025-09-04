// package pod

// import (
// 	"context"
// 	"strings"

// 	"AtlHyper/atlhyper_master/interfaces/datasource"
// )

// func BuildPodOverview(ctx context.Context, clusterID string) (*PodOverviewDTO, error) {
//     pods, err := datasource.GetPodListLatest(ctx, clusterID)
//     if err != nil {
//         return nil, err
//     }

//     var running, pending, failed, unknown int
//     items := make([]PodOverviewItem, 0, len(pods))

//     for _, p := range pods {
//         switch strings.ToLower(p.Summary.Phase) {
//         case "running":
//             running++
//         case "pending":
//             pending++
//         case "failed":
//             failed++
//         default:
//             unknown++
//         }

//         // Deployment 名（ControlledBy）
//         deployment := ""
//         if p.Summary.ControlledBy != nil {
//             deployment = p.Summary.ControlledBy.Name
//         }

//         // Pod Metrics
//         var (
//             cpu    string
//             cpuPct float64
//             mem    string
//             memPct float64
//         )
//         if p.Metrics != nil {
//             cpu = p.Metrics.CPU.Usage
//             cpuPct = p.Metrics.CPU.UtilPct
//             mem = p.Metrics.Memory.Usage
//             memPct = p.Metrics.Memory.UtilPct
//         }

//         items = append(items, PodOverviewItem{
//             Namespace:  p.Summary.Namespace,
//             Deployment: deployment,
//             Name:       p.Summary.Name,
//             Ready:      p.Summary.Ready,
//             Phase:      p.Summary.Phase,
//             Restarts:   p.Summary.Restarts,
//             CPU:        cpu,
//             CPUPercent: cpuPct,
//             Memory:     mem,
//             MemPercent: memPct,
//             StartTime:  p.Summary.StartTime,
//             Node:       p.Summary.Node,
//         })
//     }

//     return &PodOverviewDTO{
//         Cards: PodCards{
//             Running: running,
//             Pending: pending,
//             Failed:  failed,
//             Unknown: unknown,
//         },
//         Pods: items,
//     }, nil
// }

// interfaces/ui_api/pod/overview.go
package pod

import (
	"AtlHyper/atlhyper_master/interfaces/datasource"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func BuildPodOverview(ctx context.Context, clusterID string) (*PodOverviewDTO, error) {
    pods, err := datasource.GetPodListLatest(ctx, clusterID)
    if err != nil {
        return nil, err
    }

    var running, pending, failed, unknown int
    items := make([]PodOverviewItem, 0, len(pods))

    for _, p := range pods {
        switch strings.ToLower(p.Summary.Phase) {
        case "running":
            running++
        case "pending":
            pending++
        case "failed":
            failed++
        default:
            unknown++
        }

        deployment := ""
        if p.Summary.ControlledBy != nil {
            deployment = p.Summary.ControlledBy.Name
        }

        var (
            cpuCores float64 // 数值：core
            cpuPct   float64 // 数值：%
            memM     int     // 数值：m≈Mi
            memPct   float64 // 数值：%
            cpuText, cpuPctText, memText, memPctText string // 展示文本
        )

        if p.Metrics != nil {
            // 1) 解析使用量
            mCPU := parseCPUToMilli(p.Metrics.CPU.Usage) // "1m" → 1；"0.5" → 500
            cpuCores = float64(mCPU) / 1000.0            // 数值字段仍按 core 保留
            memBytes := parseMemToBytes(p.Metrics.Memory.Usage)
            memM = bytesToM(memBytes)

            // 2) 百分比（Agent 已算好）。这里统一保留三位小数
            cpuPct = roundTo(p.Metrics.CPU.UtilPct, 3)
            memPct = roundTo(p.Metrics.Memory.UtilPct, 3)

            // 3) 展示文本（带单位/百分号）
            cpuText = fmt.Sprintf("%dm", mCPU)
            cpuPctText = formatPct(cpuPct)   // "0.100%"
            memText = fmt.Sprintf("%d m", memM)
            memPctText = formatPct(memPct)   // "2.600%"
        } else {
            // 无 metrics：回退文本
            cpuText = "0m"
            cpuPctText = "—"
            memText = "0 m"
            memPctText = "—"
        }

        items = append(items, PodOverviewItem{
            Namespace:  p.Summary.Namespace,
            Deployment: deployment,
            Name:       p.Summary.Name,
            Ready:      p.Summary.Ready,
            Phase:      p.Summary.Phase,
            Restarts:   p.Summary.Restarts,

            CPU:        cpuCores,  // core（用于排序/阈值）
            CPUPercent: cpuPct,    // 数值（用于图表/计算）
            Memory:     memM,      // m≈Mi（用于排序/阈值）
            MemPercent: memPct,    // 数值

            CPUText:        cpuText,        // "1m"
            CPUPercentText: cpuPctText,     // "0.100%"
            MemoryText:     memText,        // "13 m"
            MemPercentText: memPctText,     // "2.600%"

            StartTime:  p.Summary.StartTime,
            Node:       p.Summary.Node,
        })
    }

    return &PodOverviewDTO{
        Cards: PodCards{Running: running, Pending: pending, Failed: failed, Unknown: unknown},
        Pods:  items,
    }, nil
}

// ===== helpers =====

// "0"/"125m"/"0.5" → m（millicores）
func parseCPUToMilli(s string) int64 {
    s = strings.TrimSpace(s)
    if s == "" {
        return 0
    }
    if strings.HasSuffix(s, "m") {
        v := strings.TrimSuffix(s, "m")
        n, err := strconv.ParseInt(v, 10, 64)
        if err != nil {
            return 0
        }
        return n
    }
    // 其余按 core（可含小数）
    f, err := strconv.ParseFloat(s, 64)
    if err != nil {
        return 0
    }
    return int64(math.Round(f * 1000.0))
}

// "10088Ki"/"220Mi"/"1Gi" → bytes
func parseMemToBytes(s string) int64 {
    s = strings.TrimSpace(s)
    if s == "" {
        return 0
    }
    // 拆数字/单位
    i := len(s)
    for i > 0 && (s[i-1] < '0' || s[i-1] > '9') {
        i--
    }
    num, unit := s[:i], strings.ToUpper(strings.TrimSpace(s[i:]))
    f, err := strconv.ParseFloat(num, 64)
    if err != nil {
        return 0
    }
    switch unit {
    case "KI":
        return int64(f * 1024)
    case "MI":
        return int64(f * 1024 * 1024)
    case "GI":
        return int64(f * 1024 * 1024 * 1024)
    case "TI":
        return int64(f * 1024 * 1024 * 1024 * 1024)
    case "", "B":
        return int64(f)
    default:
        return int64(f) // 未识别单位，按字节
    }
}

// bytes → m（≈Mi），四舍五入为整数
func bytesToM(bytes int64) int {
    if bytes <= 0 {
        return 0
    }
    mi := float64(bytes) / (1024.0 * 1024.0)
    return int(math.Round(mi))
}

// 四舍五入到 n 位小数
func roundTo(x float64, n int) float64 {
    if n <= 0 {
        return math.Round(x)
    }
    p := math.Pow(10, float64(n))
    return math.Round(x*p) / p
}

// 0-100 的百分比数值 → 文本（保留三位小数并带 %）
func formatPct(x float64) string {
    // 确保三位小数：如 0 → "0.000%"
    return fmt.Sprintf("%.3f%%", x)
}
