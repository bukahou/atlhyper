package cluster

import (
	readonlypod "NeuroController/internal/readonly/pod"
	modelpod "NeuroController/model/pod"
	"context"
)

func PodList(ctx context.Context) ([]modelpod.Pod, error) {
	m, err := readonlypod.ListPods(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}