// internal/readonly/deployment/convert_helpers.go
package deployment

import (
	modeldep "AtlHyper/model/k8s"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func selectorToString(sel *metav1.LabelSelector) string {
	if sel == nil {
		return ""
	}
	// 简化：仅拉平 MatchLabels；MatchExpressions 可按需追加
	if len(sel.MatchLabels) == 0 {
		return ""
	}
	// 稳定顺序可按需排序；这里直接拼
	s := ""
	sep := ""
	for k, v := range sel.MatchLabels {
		s += sep + k + "=" + v
		sep = ","
	}
	return s
}

func toLabelSelector(sel *metav1.LabelSelector) modeldep.LabelSelector {
	if sel == nil {
		return modeldep.LabelSelector{}
	}
	out := modeldep.LabelSelector{
		MatchLabels: map[string]string{},
	}
	for k, v := range sel.MatchLabels {
		out.MatchLabels[k] = v
	}
	for _, e := range sel.MatchExpressions {
		out.MatchExpressions = append(out.MatchExpressions, modeldep.LabelExpr{
			Key:      e.Key,
			Operator: string(e.Operator),
			Values:   append([]string(nil), e.Values...),
		})
	}
	return out
}

func toStrategy(s *appsv1.DeploymentStrategy) *modeldep.Strategy {
	if s == nil {
		return nil
	}
	out := &modeldep.Strategy{Type: string(s.Type)}
	if s.Type == appsv1.RollingUpdateDeploymentStrategyType && s.RollingUpdate != nil {
		out.RollingUpdate = &modeldep.RollingUpdateStrategy{
			MaxUnavailable: intOrStrToString(s.RollingUpdate.MaxUnavailable),
			MaxSurge:       intOrStrToString(s.RollingUpdate.MaxSurge),
		}
	}
	return out
}

func intOrStrToString(p *intstr.IntOrString) string {
	if p == nil {
		return ""
	}
	if p.Type == intstr.String {
		return p.StrVal
	}
	return fmtInt32(int32(p.IntValue()))
}

func fmtInt32(v int32) string { return strconvItoa(int(v)) }
func strconvItoa(i int) string {
	// 避免额外导入 strconv；小工具
	return fmt.Sprintf("%d", i)
}

func toConditions(conds []appsv1.DeploymentCondition) []modeldep.Condition {
	if len(conds) == 0 {
		return nil
	}
	out := make([]modeldep.Condition, 0, len(conds))
	for _, c := range conds {
		out = append(out, modeldep.Condition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
			LastUpdateTime:     c.LastUpdateTime.Time,
			LastTransitionTime: c.LastTransitionTime.Time,
		})
	}
	return out
}

// deriveRollout —— 基于 Deployment 状态与条件推导一个友好的 Rollout 概览
func deriveRollout(d *appsv1.Deployment) *modeldep.Rollout {
	replicas := valueOrDefaultInt32(d.Spec.Replicas, 1)

	// Paused
	if d.Spec.Paused {
		return &modeldep.Rollout{Phase: "Paused", Badges: []string{"Paused"}}
	}

	// ProgressDeadlineExceeded
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Status == "False" && c.Reason == "ProgressDeadlineExceeded" {
			return &modeldep.Rollout{Phase: "Degraded", Message: c.Message, Badges: []string{"Timeout"}}
		}
	}

	// 完成：全部副本已更新且可用
	if d.Status.UpdatedReplicas == replicas && d.Status.AvailableReplicas == replicas && d.Generation <= d.Status.ObservedGeneration {
		return &modeldep.Rollout{Phase: "Complete"}
	}

	// 进行中
	return &modeldep.Rollout{Phase: "Progressing"}
}

// 注解/标签精简
func pickDeployAnnotations(ann map[string]string) map[string]string {
	if len(ann) == 0 {
		return nil
	}
	keys := []string{
		"deployment.kubernetes.io/revision",
		"kubectl.kubernetes.io/last-applied-configuration",
	}
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := ann[k]; ok && v != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func pickPodTemplateAnnotations(ann map[string]string) map[string]string {
	if len(ann) == 0 {
		return nil
	}
	// 根据实际需要挑选，避免体积过大；此处示例为空保持扩展点
	return nil
}

func copyStrMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func strPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
