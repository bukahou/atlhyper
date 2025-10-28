package service

import (
	"AtlHyper/atlhyper_aiservice/client/master"
	masterModel "AtlHyper/atlhyper_master/aiservice/model"
	m "AtlHyper/model/event"
	"context"
	"encoding/json"
	"fmt"
)

func RunStage2FetchContext(ctx context.Context, clusterID string, stage1 map[string]interface{}, events []m.EventLog) (map[string]interface{}, error) {
	var need struct {
		Pods        []struct{ Namespace, Name string } `json:"pods"`
		Deployments []struct{ Namespace, Name string } `json:"deployments"`
		Services    []struct{ Namespace, Name string } `json:"services"`
		Nodes       []string                           `json:"nodes"`
	}
	if j, ok := stage1["ai_json"].(map[string]interface{}); ok {
		if nr, ok := j["needResources"]; ok {
			b, _ := json.Marshal(nr)
			_ = json.Unmarshal(b, &need)
		}
	}

	// 转换为 master 的结构
	req := &masterModel.AIFetchRequest{
		ClusterID:   clusterID,
		Pods:        toRef(need.Pods),
		Deployments: toRef(need.Deployments),
		Services:    toRef(need.Services),
		Nodes:       need.Nodes,
	}
	resp, err := master.FetchAIContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch context failed: %v", err)
	}

	return map[string]interface{}{
		"need":  need,
		"fetch": resp,
	}, nil
}

func toRef(in []struct{ Namespace, Name string }) []masterModel.ResourceRef {
	out := make([]masterModel.ResourceRef, 0, len(in))
	for _, x := range in {
		out = append(out, masterModel.ResourceRef{Namespace: x.Namespace, Name: x.Name})
	}
	return out
}
