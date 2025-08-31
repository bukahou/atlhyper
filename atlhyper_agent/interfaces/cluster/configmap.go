package cluster

import (
	readonlyconfigmap "AtlHyper/atlhyper_agent/internal/readonly/configmap"
	modelconfigmap "AtlHyper/model/configmap"
	"context"
)

func ConfigMapList(ctx context.Context) ([]modelconfigmap.ConfigMap, error) {
	m, err := readonlyconfigmap.ListConfigMaps(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}