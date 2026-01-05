// atlhyper_master/service/deployment/deployment_detail.go
package deployment

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master/model/ui"
	"AtlHyper/atlhyper_master/repository"
	mod "AtlHyper/model/k8s"
)

// BuildDeploymentDetail —— 根据 clusterID + namespace + name 返回详情
func BuildDeploymentDetail(ctx context.Context, clusterID, namespace, name string) (*ui.DeploymentDetailDTO, error) {
	list, err := repository.GetDeploymentListLatest(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("get deployment list failed: %w", err)
	}
	for _, d := range list {
		if d.Summary.Namespace == namespace && d.Summary.Name == name {
			dto := fromModelToDetail(d)
			return &dto, nil
		}
	}
	return nil, fmt.Errorf("deployment not found: %s/%s (cluster=%s)", namespace, name, clusterID)
}

func fromModelToDetail(d mod.Deployment) ui.DeploymentDetailDTO {
	out := ui.DeploymentDetailDTO{
		// summary
		Name:        d.Summary.Name,
		Namespace:   d.Summary.Namespace,
		Strategy:    d.Summary.Strategy,
		Replicas:    d.Summary.Replicas,
		Updated:     d.Summary.Updated,
		Ready:       d.Summary.Ready,
		Available:   d.Summary.Available,
		Unavailable: d.Summary.Unavailable,
		Paused:      d.Summary.Paused,
		Selector:    d.Summary.Selector,
		CreatedAt:   d.Summary.CreatedAt,
		Age:         d.Summary.Age,

		// template
		Template: ui.DeploymentTemplateDTO{
			Labels:             d.Template.Labels,
			Annotations:        d.Template.Annotations,
			Containers:         d.Template.Containers,
			Volumes:            d.Template.Volumes,
			ServiceAccountName: d.Template.ServiceAccountName,
			NodeSelector:       d.Template.NodeSelector,
			HostNetwork:        d.Template.HostNetwork,
			DNSPolicy:          d.Template.DNSPolicy,
			RuntimeClassName:   d.Template.RuntimeClassName,
			ImagePullSecrets:   d.Template.ImagePullSecrets,
		},

		// status
		Status: ui.DeploymentStatusDTO{
			ObservedGeneration:  d.Status.ObservedGeneration,
			Replicas:            d.Status.Replicas,
			UpdatedReplicas:     d.Status.UpdatedReplicas,
			ReadyReplicas:       d.Status.ReadyReplicas,
			AvailableReplicas:   d.Status.AvailableReplicas,
			UnavailableReplicas: d.Status.UnavailableReplicas,
			CollisionCount:      d.Status.CollisionCount,
		},

		Labels:      d.Labels,
		Annotations: d.Annotations,
	}

	// spec
	out.Spec = ui.DeploymentSpecDTO{
		Replicas:                d.Spec.Replicas,
		MinReadySeconds:         d.Spec.MinReadySeconds,
		RevisionHistoryLimit:    d.Spec.RevisionHistoryLimit,
		ProgressDeadlineSeconds: d.Spec.ProgressDeadlineSeconds,
		StrategyType:            "",
		MatchLabels:             d.Spec.Selector.MatchLabels,
	}
	if d.Spec.Strategy != nil {
		out.Spec.StrategyType = d.Spec.Strategy.Type
		if d.Spec.Strategy.RollingUpdate != nil {
			out.Spec.MaxUnavailable = d.Spec.Strategy.RollingUpdate.MaxUnavailable
			out.Spec.MaxSurge = d.Spec.Strategy.RollingUpdate.MaxSurge
		}
	}

	// conditions
	if len(d.Status.Conditions) > 0 {
		out.Conditions = make([]ui.DeploymentCondDTO, 0, len(d.Status.Conditions))
		for _, c := range d.Status.Conditions {
			out.Conditions = append(out.Conditions, ui.DeploymentCondDTO{
				Type:               c.Type,
				Status:             c.Status,
				Reason:             c.Reason,
				Message:            c.Message,
				LastUpdateTime:     c.LastUpdateTime,
				LastTransitionTime: c.LastTransitionTime,
			})
		}
	}

	// rollout
	if d.Rollout != nil {
		out.Rollout = &ui.DeploymentRolloutDTO{
			Phase:   d.Rollout.Phase,
			Message: d.Rollout.Message,
			Badges:  d.Rollout.Badges,
		}
	}

	// replicaSets
	if len(d.ReplicaSets) > 0 {
		out.ReplicaSets = make([]ui.ReplicaSetBriefDTO, 0, len(d.ReplicaSets))
		for _, rs := range d.ReplicaSets {
			out.ReplicaSets = append(out.ReplicaSets, ui.ReplicaSetBriefDTO{
				Name:      rs.Name,
				Namespace: rs.Namespace,
				Revision:  rs.Revision,
				Replicas:  rs.Replicas,
				Ready:     rs.Ready,
				Available: rs.Available,
				CreatedAt: rs.CreatedAt,
				Age:       rs.Age,
			})
		}
	}

	return out
}
