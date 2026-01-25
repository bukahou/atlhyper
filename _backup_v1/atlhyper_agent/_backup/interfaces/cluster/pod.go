package cluster

import (
	readonlypod "AtlHyper/atlhyper_agent/internal/readonly/pod"
	modelpod "AtlHyper/model/k8s"
	"context"
)

func PodList(ctx context.Context) ([]modelpod.Pod, error) {
	m, err := readonlypod.ListPods(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}