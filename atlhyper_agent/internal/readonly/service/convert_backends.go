package service

import (
	"context"
	"time"

	modelsvc "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// buildBackendIndexFromSlices —— 一次性拉取全量 EndpointSlice，构建 ns/name → backends 的索引
// 返回：索引、聚合的 slice 数量、错误（若 API 不可用/鉴权失败等）
func buildBackendIndexFromSlices(ctx context.Context, cs *kubernetes.Clientset) (map[string]modelsvc.ServiceBackends, int, error) {
	slices, err := cs.DiscoveryV1().EndpointSlices(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil || slices == nil {
		return map[string]modelsvc.ServiceBackends{}, 0, err
	}
	out := make(map[string]modelsvc.ServiceBackends, 256)
	now := time.Now()

	for i := range slices.Items {
		es := &slices.Items[i]
		svcName := es.Labels["kubernetes.io/service-name"]
		if svcName == "" {
			continue
		}
		key := es.Namespace + "/" + svcName

		ports := mergeSlicePorts(es.Ports)
		ready, notReady, total, flat := flattenEndpointsFromSlice(es.Endpoints)

		be := out[key]
		be.Summary.Ready += ready
		be.Summary.NotReady += notReady
		be.Summary.Total += total
		be.Summary.Slices++
		be.Summary.Updated = now
		be.Ports = mergePortDefs(be.Ports, ports)
		be.Endpoints = append(be.Endpoints, flat...)
		out[key] = be
	}
	return out, len(slices.Items), nil
}

// buildBackendIndexFromEndpoints —— 兜底方案：使用 v1/Endpoints 构建 ns/name → backends
func buildBackendIndexFromEndpoints(ctx context.Context, cs *kubernetes.Clientset) (map[string]modelsvc.ServiceBackends, int, error) {
	eps, err := cs.CoreV1().Endpoints(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil || eps == nil {
		return map[string]modelsvc.ServiceBackends{}, 0, err
	}
	out := make(map[string]modelsvc.ServiceBackends, 256)
	now := time.Now()

	for i := range eps.Items {
		ep := &eps.Items[i]
		key := ep.Namespace + "/" + ep.Name
		ready, notReady, total, flat := flattenEndpointsFromEndpoints(ep.Subsets)
		ports := portsFromEndpoints(ep.Subsets)

		be := out[key]
		be.Summary.Ready += ready
		be.Summary.NotReady += notReady
		be.Summary.Total += total
		be.Summary.Updated = now
		be.Ports = mergePortDefs(be.Ports, ports)
		be.Endpoints = append(be.Endpoints, flat...)
		out[key] = be
	}
	return out, len(eps.Items), nil
}

// ---------- EndpointSlice → 扁平端点 ----------

func mergeSlicePorts(ports []discoveryv1.EndpointPort) []modelsvc.EndpointPort {
	if len(ports) == 0 {
		return nil
	}
	out := make([]modelsvc.EndpointPort, 0, len(ports))
	for _, p := range ports {
		if p.Port == nil {
			continue
		}
		out = append(out, modelsvc.EndpointPort{
			Name:        stringPtrValue(p.Name),
			Port:        *p.Port,
			Protocol:    protoPtrValue(p.Protocol),
			AppProtocol: stringPtrValue(p.AppProtocol),
		})
	}
	return out
}

func flattenEndpointsFromSlice(eps []discoveryv1.Endpoint) (ready, notReady, total int, flat []modelsvc.BackendEndpoint) {
	for _, e := range eps {
		isReady := true
		if e.Conditions.Ready != nil {
			isReady = *e.Conditions.Ready
		}
		addrs := append([]string(nil), e.Addresses...)
		total += len(addrs)
		if isReady {
			ready += len(addrs)
		} else {
			notReady += len(addrs)
		}
		for _, addr := range addrs {
			flat = append(flat, modelsvc.BackendEndpoint{
				Address:   addr,
				Ready:     isReady,
				NodeName:  stringPtrValue(e.NodeName),
				Zone:      stringPtrValue(e.Zone),
				TargetRef: toK8sRef(e.TargetRef),
			})
		}
	}
	return
}

// ---------- Endpoints → 扁平端点（兜底） ----------

func flattenEndpointsFromEndpoints(subsets []corev1.EndpointSubset) (ready, notReady, total int, flat []modelsvc.BackendEndpoint) {
	for _, ss := range subsets {
		for _, a := range ss.Addresses {
			ready++
			total++
			flat = append(flat, modelsvc.BackendEndpoint{
				Address:   a.IP,
				Ready:     true,
				NodeName:  stringPtrValue(a.NodeName), // *string → string
				TargetRef: toK8sRef(a.TargetRef),
			})
		}
		for _, a := range ss.NotReadyAddresses {
			notReady++
			total++
			flat = append(flat, modelsvc.BackendEndpoint{
				Address:   a.IP,
				Ready:     false,
				NodeName:  stringPtrValue(a.NodeName), // *string → string
				TargetRef: toK8sRef(a.TargetRef),
			})
		}
	}
	return
}

func portsFromEndpoints(subsets []corev1.EndpointSubset) []modelsvc.EndpointPort {
	type key struct {
		name, proto, app string
		port             int32
	}
	uniq := map[key]struct{}{}
	var out []modelsvc.EndpointPort
	for _, ss := range subsets {
		for _, p := range ss.Ports {
			k := key{
				name:  p.Name,
				proto: string(p.Protocol),
				app:   stringPtrValue(p.AppProtocol),
				port:  p.Port,
			}
			if _, ok := uniq[k]; ok {
				continue
			}
			uniq[k] = struct{}{}
			out = append(out, modelsvc.EndpointPort{
				Name:        p.Name,
				Port:        p.Port,
				Protocol:    string(p.Protocol),
				AppProtocol: stringPtrValue(p.AppProtocol),
			})
		}
	}
	return out
}

// ---------- 合并端口定义（去重并保持已有顺序优先） ----------

func mergePortDefs(base, add []modelsvc.EndpointPort) []modelsvc.EndpointPort {
	if len(add) == 0 {
		return base
	}
	type key struct {
		name, proto, app string
		port             int32
	}
	seen := map[key]struct{}{}
	for _, p := range base {
		seen[key{p.Name, p.Protocol, p.AppProtocol, p.Port}] = struct{}{}
	}
	for _, p := range add {
		k := key{p.Name, p.Protocol, p.AppProtocol, p.Port}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		base = append(base, p)
	}
	return base
}
