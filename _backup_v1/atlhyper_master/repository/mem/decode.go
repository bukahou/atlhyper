// repository/mem/decode.go
// Payload 解码函数
package mem

import (
	"encoding/json"
	"fmt"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/model/collect"
	"AtlHyper/model/k8s"
	"AtlHyper/model/transport"
)

// unmarshalList 通用列表解码（支持纯数组和包裹对象）
func unmarshalList[T any](raw json.RawMessage, key1, key2 string) ([]T, error) {
	var arr []T
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err == nil {
		if v, ok := m[key1]; ok {
			if err := json.Unmarshal(v, &arr); err == nil {
				return arr, nil
			}
		}
		if v, ok := m[key2]; ok {
			if err := json.Unmarshal(v, &arr); err == nil {
				return arr, nil
			}
		}
	}

	return nil, fmt.Errorf("unsupported list payload shape (expect [] or {%q|%q:[...]})", key1, key2)
}

// ============================================================
// K8s 资源解码
// ============================================================

func decodePodList(raw json.RawMessage) ([]repository.Pod, error) {
	return unmarshalList[k8s.Pod](raw, "pods", "items")
}

func decodeNodeList(raw json.RawMessage) ([]repository.Node, error) {
	return unmarshalList[k8s.Node](raw, "nodes", "items")
}

func decodeServiceList(raw json.RawMessage) ([]repository.Service, error) {
	return unmarshalList[k8s.Service](raw, "services", "items")
}

func decodeNamespaceList(raw json.RawMessage) ([]repository.Namespace, error) {
	return unmarshalList[k8s.Namespace](raw, "namespaces", "items")
}

func decodeIngressList(raw json.RawMessage) ([]repository.Ingress, error) {
	return unmarshalList[k8s.Ingress](raw, "ingresses", "items")
}

func decodeDeploymentList(raw json.RawMessage) ([]repository.Deployment, error) {
	return unmarshalList[k8s.Deployment](raw, "deployments", "items")
}

func decodeConfigMapList(raw json.RawMessage) ([]repository.ConfigMap, error) {
	return unmarshalList[k8s.ConfigMap](raw, "configmaps", "items")
}

// ============================================================
// 指标和事件解码
// ============================================================

func decodeMetricsBatch(raw json.RawMessage) ([]repository.NodeMetricsSnapshot, error) {
	var arr []collect.NodeMetricsSnapshot
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	var wrap struct {
		Snapshots []collect.NodeMetricsSnapshot `json:"snapshots"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Snapshots) > 0 {
		return wrap.Snapshots, nil
	}

	return nil, nil
}

func decodeEvents(raw json.RawMessage) ([]repository.LogEvent, error) {
	var arr []transport.LogEvent
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	var wrap struct {
		Events []transport.LogEvent `json:"events"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Events) > 0 {
		return wrap.Events, nil
	}

	return nil, nil
}
