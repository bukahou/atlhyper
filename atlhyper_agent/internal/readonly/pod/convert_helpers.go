package pod

import (
	"fmt"
	"sort"
	"time"

	modelpod "AtlHyper/model/pod"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func readyCount(statuses []corev1.ContainerStatus, total int) (ready int, all int) {
	all = total
	for _, s := range statuses {
		if s.Ready {
			ready++
		}
	}
	return
}

func totalRestarts(statuses []corev1.ContainerStatus) int {
	sum := 0
	for _, s := range statuses {
		sum += int(s.RestartCount)
	}
	return sum
}

func firstOwner(ors []metav1.OwnerReference) *modelpod.Owner {
	if len(ors) == 0 {
		return nil
	}
	// 优先 Controller=true
	sort.SliceStable(ors, func(i, j int) bool {
		ci := ors[i].Controller != nil && *ors[i].Controller
		cj := ors[j].Controller != nil && *ors[j].Controller
		if ci == cj {
			return i < j
		}
		return ci && !cj
	})
	return &modelpod.Owner{Kind: ors[0].Kind, Name: ors[0].Name}
}

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

func stringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
