// interfaces/datasource/hub_decode.go
package datasource

import (
	"encoding/json"
	"fmt"

	modcm "AtlHyper/model/configmap"
	moddep "AtlHyper/model/deployment"
	base "AtlHyper/model/event"
	moding "AtlHyper/model/ingress"
	modmetrics "AtlHyper/model/metrics"
	modns "AtlHyper/model/namespace"
	modnode "AtlHyper/model/node"
	modpod "AtlHyper/model/pod"
	modsvc "AtlHyper/model/service"
)

// ====================== 类型别名（直接复用底层模型） ======================

type Pod = modpod.Pod
type Node = modnode.Node
type Service = modsvc.Service
type Namespace = modns.Namespace
type Ingress = moding.Ingress
type Deployment = moddep.Deployment
type ConfigMap = modcm.ConfigMap

type LogEvent = base.LogEvent
type NodeMetricsSnapshot = modmetrics.NodeMetricsSnapshot

// ====================== 通用助手：列表多形状解码 ======================
// 先尝试把 raw 解成 []T；失败则尝试对象包裹里指定的 key（如 pods/items）。
func unmarshalList[T any](raw json.RawMessage, key1, key2 string) ([]T, error) {
	// A) 纯数组：[]T
	var arr []T
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	// B) 包裹对象：{ key1: [...] } 或 { key2: [...] }
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

// ====================== 解码函数（Payload → 结构体） ======================
// 说明：以下函数改为“多形状兼容”，与 metrics/events 的思路一致。

func decodePodList(raw json.RawMessage) ([]Pod, error) {
	// 兼容：[] / {"pods":[...]} / {"items":[...]}
	return unmarshalList[modpod.Pod](raw, "pods", "items")
}

func decodeNodeList(raw json.RawMessage) ([]Node, error) {
	// 兼容：[] / {"nodes":[...]} / {"items":[...]}
	return unmarshalList[modnode.Node](raw, "nodes", "items")
}

func decodeServiceList(raw json.RawMessage) ([]Service, error) {
	// 兼容：[] / {"services":[...]} / {"items":[...]}
	return unmarshalList[modsvc.Service](raw, "services", "items")
}

func decodeNamespaceList(raw json.RawMessage) ([]Namespace, error) {
	// 兼容：[] / {"namespaces":[...]} / {"items":[...]}
	return unmarshalList[modns.Namespace](raw, "namespaces", "items")
}

func decodeIngressList(raw json.RawMessage) ([]Ingress, error) {
	// 兼容：[] / {"ingresses":[...]} / {"items":[...]}
	return unmarshalList[moding.Ingress](raw, "ingresses", "items")
}

func decodeDeploymentList(raw json.RawMessage) ([]Deployment, error) {
	// 兼容：[] / {"deployments":[...]} / {"items":[...]}
	return unmarshalList[moddep.Deployment](raw, "deployments", "items")
}

func decodeConfigMapList(raw json.RawMessage) ([]ConfigMap, error) {
	// 兼容：[] / {"configmaps":[...]} / {"items":[...]}
	return unmarshalList[modcm.ConfigMap](raw, "configmaps", "items")
}

// ====================== 特殊解码：Metrics & Events（保持你现有成功实现） ======================

func decodeMetricsBatch(raw json.RawMessage) ([]NodeMetricsSnapshot, error) {
	// A. 先尝试纯数组：[]NodeMetricsSnapshot
	var arr []modmetrics.NodeMetricsSnapshot
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	// B. 再尝试包裹对象：{ "snapshots": [...] }
	var wrap struct {
		Snapshots []modmetrics.NodeMetricsSnapshot `json:"snapshots"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Snapshots) > 0 {
		return wrap.Snapshots, nil
	}

	// C. 都不是，就返回空（不报错，保持上层逻辑）
	return nil, nil
}

func decodeEvents(raw json.RawMessage) ([]LogEvent, error) {
	// A. 先尝试解成纯数组：[]LogEvent
	var arr []base.LogEvent
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	// B. 再尝试解成包裹对象：{ "events": [...] , ... }
	var wrap struct {
		Events []base.LogEvent `json:"events"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Events) > 0 {
		return wrap.Events, nil
	}

	// C. 都不是，就当空（保持上层逻辑简单）
	return nil, nil
}
