package service

import (
	"fmt"
	"time"

	modelsvc "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
)

// fmtAge —— 与 Pod/Node 一致的“简洁时长”字符串
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

func toStrSlice(fams []corev1.IPFamily) []string {
	if len(fams) == 0 {
		return nil
	}
	out := make([]string, 0, len(fams))
	for _, f := range fams {
		out = append(out, string(f))
	}
	return out
}

func stringPtrValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func protoPtrValue(p *corev1.Protocol) string {
	if p == nil {
		return string(corev1.ProtocolTCP)
	}
	return string(*p)
}

func strPtrValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// fmtInt32 —— 仅用于 TargetPort 是 Int 的情况
func fmtInt32(v int32) string {
	return fmt.Sprintf("%d", v)
}

// toK8sRef —— 将 Kubernetes 的对象引用转为精简引用（可能为 nil）
func toK8sRef(ref *corev1.ObjectReference) *modelsvc.K8sRef {
	if ref == nil {
		return nil
	}
	out := &modelsvc.K8sRef{
		Kind:      ref.Kind,
		Namespace: ref.Namespace,
		Name:      ref.Name,
		UID:       string(ref.UID),
	}
	// 将空值标准化为空字符串（可选）
	if out.Kind == "" && out.Namespace == "" && out.Name == "" && out.UID == "" {
		return nil
	}
	return out
}

// 专用于 *corev1.IPFamilyPolicy → string
func ipFamilyPolicyPtrValue(p *corev1.IPFamilyPolicy) string {
	if p == nil {
		return ""
	}
	return string(*p)
}

// 专用于 *corev1.ServiceInternalTrafficPolicy → string
func internalTrafficPolicyPtrValue(p *corev1.ServiceInternalTrafficPolicy) string {
	if p == nil {
		return ""
	}
	return string(*p)
}
