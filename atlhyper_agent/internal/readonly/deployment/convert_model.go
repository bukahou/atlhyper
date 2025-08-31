// internal/readonly/deployment/convert_model.go
package deployment

import (
	"fmt"
	"time"

	modeldep "AtlHyper/model/deployment"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

func buildModel(d *appsv1.Deployment, rsIdx map[types.UID][]appsv1.ReplicaSet) modeldep.Deployment {
	created := d.CreationTimestamp.Time

	// Summary
	sum := modeldep.DeploymentSummary{
		Name:        d.Name,
		Namespace:   d.Namespace,
		Strategy:    string(d.Spec.Strategy.Type),
		Replicas:    valueOrDefaultInt32(d.Spec.Replicas, 1),
		Updated:     d.Status.UpdatedReplicas,
		Ready:       d.Status.ReadyReplicas,
		Available:   d.Status.AvailableReplicas,
		Unavailable: d.Status.UnavailableReplicas,
		Paused:      d.Spec.Paused,
		CreatedAt:   created,
		Age:         fmtAge(created),
		Selector:    selectorToString(d.Spec.Selector),
	}

	// Spec
	spec := modeldep.DeploymentSpec{
		Replicas:                 d.Spec.Replicas,
		Selector:                 toLabelSelector(d.Spec.Selector),
		Strategy:                 toStrategy(&d.Spec.Strategy),
		MinReadySeconds:          d.Spec.MinReadySeconds,
		RevisionHistoryLimit:     d.Spec.RevisionHistoryLimit,
		ProgressDeadlineSeconds:  d.Spec.ProgressDeadlineSeconds,
	}

	// Template
	tpl := toPodTemplate(&d.Spec.Template)

	// Status
	status := modeldep.DeploymentStatus{
		ObservedGeneration:  d.Status.ObservedGeneration,
		Replicas:            d.Status.Replicas,
		UpdatedReplicas:     d.Status.UpdatedReplicas,
		ReadyReplicas:       d.Status.ReadyReplicas,
		AvailableReplicas:   d.Status.AvailableReplicas,
		UnavailableReplicas: d.Status.UnavailableReplicas,
		CollisionCount:      d.Status.CollisionCount,
		Conditions:          toConditions(d.Status.Conditions),
	}

	roll := deriveRollout(d)

	// ReplicaSets（可选）
	var briefs []modeldep.ReplicaSetBrief
	if rsIdx != nil {
		briefs = rsBriefs(d.UID, rsIdx)
	}

	return modeldep.Deployment{
		Summary:     sum,
		Spec:        spec,
		Template:    tpl,
		Status:      status,
		Rollout:     roll,
		ReplicaSets: briefs,
		Annotations: pickDeployAnnotations(d.Annotations),
		Labels:      copyStrMap(d.Labels),
	}
}

func valueOrDefaultInt32(p *int32, def int32) int32 {
	if p == nil {
		return def
	}
	return *p
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
