package node

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_agent/sdk"
	modelnode "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// —— Age 文本化
func fmtAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	day := d / (24 * time.Hour)
	d -= day * 24 * time.Hour
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	switch {
	case day > 0:
		return fmt.Sprintf("%dd%dh", day, h)
	case h > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// —— Node 条件 → 徽标（简化）
func deriveBadges(conds []corev1.NodeCondition) []string {
	var out []string
	ready := "Unknown"
	for _, c := range conds {
		switch c.Type {
		case corev1.NodeReady:
			ready = string(c.Status)
		case corev1.NodeMemoryPressure:
			if c.Status == corev1.ConditionTrue {
				out = append(out, "MemoryPressure")
			}
		case corev1.NodeDiskPressure:
			if c.Status == corev1.ConditionTrue {
				out = append(out, "DiskPressure")
			}
		case corev1.NodePIDPressure:
			if c.Status == corev1.ConditionTrue {
				out = append(out, "PIDPressure")
			}
		}
	}
	if ready != "True" {
		out = append(out, "NotReady")
	}
	// 去重略；通常徽标数量有限
	return out
}

// —— 从 ResourceList 中提取字符串资源，兼容 CPU/memory/pods/ephemeral-storage 以及自定义标量
func extractNodeResources(rl corev1.ResourceList) modelnode.NodeResources {
	nr := modelnode.NodeResources{
		ScalarResources: map[string]string{},
	}
	// 标准字段
	if q := rl.Cpu(); q != nil && !q.IsZero() {
		// 直接使用原始 String 表达：可能是“8”或“7700m”
		nr.CPU = q.String()
	}
	if q := rl.Memory(); q != nil && !q.IsZero() {
		nr.Memory = q.String()
	}
	if q, ok := rl[corev1.ResourcePods]; ok && !q.IsZero() {
		nr.Pods = q.String()
	}
	if q, ok := rl[corev1.ResourceEphemeralStorage]; ok && !q.IsZero() {
		nr.EphemeralStorage = q.String()
	}
	// 其它标量（HugePages/扩展资源等）
	for k, v := range rl {
		if k == corev1.ResourceCPU || k == corev1.ResourceMemory ||
			k == corev1.ResourcePods || k == corev1.ResourceEphemeralStorage {
			continue
		}
		if !v.IsZero() {
			nr.ScalarResources[string(k)] = v.String()
		}
	}
	if len(nr.ScalarResources) == 0 {
		nr.ScalarResources = nil
	}
	return nr
}

// —— 为 metrics 计算提供的数值基线（CPU cores / memory bytes / pods int）
func quantityCores(primary, fallback string) float64 {
	// primary 优先；为空则用 fallback
	q := parseQuantity(primary)
	if q == nil || q.IsZero() {
		q = parseQuantity(fallback)
	}
	if q == nil {
		return 0
	}
	return q.AsApproximateFloat64()
}
func quantityBytes(primary, fallback string) float64 {
	q := parseQuantity(primary)
	if q == nil || q.IsZero() {
		q = parseQuantity(fallback)
	}
	if q == nil {
		return 0
	}
	return float64(q.Value())
}
func quantityInt(primary, fallback string) int {
	q := parseQuantity(primary)
	if q == nil || q.IsZero() {
		q = parseQuantity(fallback)
	}
	if q == nil {
		return 0
	}
	return int(q.Value())
}
func parseQuantity(s string) *resource.Quantity {
	if s == "" {
		return nil
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return nil
	}
	return &q
}

// —— 将 allocatable/capacity 展示成字符串（可能 primary 为空则回退）
func normalizeCPU(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
func normalizeBytes(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

// —— 统计每个节点上"在用 Pod 数"
// 规则（可按需调整）：统计 Assigned 且未完成的 Pod（phase != Succeeded && != Failed && deletionTimestamp==nil）
func countPodsPerNode(ctx context.Context) (map[string]int, error) {
	cs := sdk.Get().CoreClient()
	pl, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, 64)
	for i := range pl.Items {
		p := &pl.Items[i]
		if p.Spec.NodeName == "" {
			continue
		}
		if p.DeletionTimestamp != nil {
			continue
		}
		if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
			continue
		}
		out[p.Spec.NodeName]++
	}
	return out, nil
}
