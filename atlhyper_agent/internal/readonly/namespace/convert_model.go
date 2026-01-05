// internal/readonly/namespace/convert_model.go
package namespace

import (
	"fmt"
	"time"

	modelns "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
)

func buildSkeleton(ns *corev1.Namespace) modelns.Namespace {
	created := ns.CreationTimestamp.Time
	phase := string(ns.Status.Phase)
	return modelns.Namespace{
		Summary: modelns.NamespaceSummary{
			Name:        ns.Name,
			Phase:       phase,
			CreatedAt:   created,
			Age:         fmtAge(created),
			Labels:      copyStrMap(ns.Labels),
			Annotations: copyStrMap(ns.Annotations),
		},
		Counts: modelns.NamespaceCounts{}, // 后续填充
	}
}

func attachBadges(dst *modelns.Namespace, quotaExceeded bool) {
	var badges []string
	if dst.Summary.Phase == "Terminating" {
		badges = append(badges, "Terminating")
	}
	if quotaExceeded {
		badges = append(badges, "QuotaExceeded")
	}
	dst.Badges = badges
}

// ===== helpers for skeleton =====

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

func copyStrMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
