// command/apply.go
// apply_manifests 指令处理 — 通过 Dynamic Client + Server-Side Apply 应用多文档 YAML
package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// ApplyManifestsResult 应用清单的结果（只在全部应用成功时返回；任何一项失败整条指令失败）
type ApplyManifestsResult struct {
	ResourceTotal   int `json:"resourceTotal"`
	ResourceChanged int `json:"resourceChanged"`
}

// applyAllowedKinds 应用清单允许的资源类型：应用自身的工作负载、网络与配置。
// ⛔ 不含集群级资源、RBAC、准入 Webhook、CRD —— 这些能把部署权限放大成整个集群的权限。
var applyAllowedKinds = map[schema.GroupKind]bool{
	{Group: "apps", Kind: "Deployment"}:                     true,
	{Group: "apps", Kind: "StatefulSet"}:                    true,
	{Group: "", Kind: "Service"}:                            true,
	{Group: "", Kind: "ConfigMap"}:                          true,
	{Group: "", Kind: "Secret"}:                             true,
	{Group: "", Kind: "ServiceAccount"}:                     true,
	{Group: "", Kind: "PersistentVolumeClaim"}:              true,
	{Group: "batch", Kind: "Job"}:                           true,
	{Group: "batch", Kind: "CronJob"}:                       true,
	{Group: "networking.k8s.io", Kind: "Ingress"}:           true,
	{Group: "networking.k8s.io", Kind: "NetworkPolicy"}:     true,
	{Group: "gateway.networking.k8s.io", Kind: "HTTPRoute"}: true,
	{Group: "autoscaling", Kind: "HorizontalPodAutoscaler"}: true,
	{Group: "policy", Kind: "PodDisruptionBudget"}:          true,
	{Group: "cilium.io", Kind: "CiliumNetworkPolicy"}:       true,
}

// applyProtectedNamespaces 应用清单不能写入的命名空间
var applyProtectedNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}

// planItem 一个要应用的对象，以及它在集群里的位置
type planItem struct {
	object    *unstructured.Unstructured
	resource  schema.GroupVersionResource
	namespace string
}

// objectApplier 对单个对象做 Server-Side Apply；dryRun 时只让 API Server 校验，不落地
type objectApplier interface {
	Apply(ctx context.Context, item planItem, dryRun bool) error
}

// planApply 核对整批清单并算出每个对象的位置。任何一项不合格整批拒绝，一个都不应用。
func planApply(objects []*unstructured.Unstructured, mapper meta.RESTMapper) ([]planItem, error) {
	if len(objects) == 0 {
		return nil, fmt.Errorf("清单里没有任何资源")
	}
	plan := make([]planItem, 0, len(objects))
	for _, obj := range objects {
		gvk := obj.GroupVersionKind()
		label := fmt.Sprintf("%s/%s", gvk.Kind, obj.GetName())
		if !applyAllowedKinds[gvk.GroupKind()] {
			return nil, fmt.Errorf("%s：资源类型 %s 不在应用清单的允许名单里", label, gvk.GroupKind())
		}
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, fmt.Errorf("%s：%w", label, err)
		}
		if mapping.Scope.Name() != meta.RESTScopeNameNamespace {
			return nil, fmt.Errorf("%s：只能应用命名空间级资源", label)
		}
		ns := obj.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		if applyProtectedNamespaces[ns] {
			return nil, fmt.Errorf("%s：不能写入命名空间 %s", label, ns)
		}
		plan = append(plan, planItem{object: obj, resource: mapping.Resource, namespace: ns})
	}
	return plan, nil
}

// runApply 先对整批预演（API Server 校验但不落地），全部通过后再逐个应用。
// 真正应用时有一项失败即返回错误，错误里带上已经应用的数量。
func runApply(ctx context.Context, plan []planItem, applier objectApplier) (int, error) {
	for _, item := range plan {
		if err := applier.Apply(ctx, item, true); err != nil {
			return 0, fmt.Errorf("预演未通过，未应用任何资源：%s/%s：%w", item.object.GetKind(), item.object.GetName(), err)
		}
	}
	applied := 0
	for _, item := range plan {
		if err := applier.Apply(ctx, item, false); err != nil {
			return applied, fmt.Errorf("应用中断（已应用 %d/%d）：%s/%s：%w", applied, len(plan), item.object.GetKind(), item.object.GetName(), err)
		}
		applied++
	}
	return applied, nil
}

// dynamicApplier 用 dynamic client 做 Server-Side Apply（Force 接管其他 manager 的字段）
type dynamicApplier struct {
	client dynamic.Interface
}

func (a dynamicApplier) Apply(ctx context.Context, item planItem, dryRun bool) error {
	data, err := json.Marshal(item.object.Object)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	opts := metav1.PatchOptions{FieldManager: "atlhyper-agent", Force: boolPtr(true)}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}
	_, err = a.client.Resource(item.resource).Namespace(item.namespace).
		Patch(ctx, item.object.GetName(), types.ApplyPatchType, data, opts)
	return err
}

// handleApplyManifests applies multi-doc YAML manifests using Server-Side Apply
func (s *commandService) handleApplyManifests(ctx context.Context, params map[string]any) (string, error) {
	manifests, ok := params["manifests"].(string)
	if !ok || strings.TrimSpace(manifests) == "" {
		return "", fmt.Errorf("missing or empty 'manifests' parameter")
	}

	// Get dynamic client and discovery client from the K8s client
	dynClient, discClient, err := s.getDynamicClients()
	if err != nil {
		return "", fmt.Errorf("failed to get dynamic client: %w", err)
	}

	// Parse multi-doc YAML
	objects, err := parseMultiDocYAML(manifests)
	if err != nil {
		return "", fmt.Errorf("failed to parse manifests: %w", err)
	}

	// Build REST mapper for GVR resolution
	groupResources, err := restmapper.GetAPIGroupResources(discClient)
	if err != nil {
		return "", fmt.Errorf("failed to get API group resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	plan, err := planApply(objects, mapper)
	if err != nil {
		return "", err
	}
	applied, err := runApply(ctx, plan, dynamicApplier{client: dynClient})
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(ApplyManifestsResult{ResourceTotal: len(plan), ResourceChanged: applied})
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

func boolPtr(b bool) *bool { return &b }

// getDynamicClients creates dynamic and discovery clients from the K8s REST config
func (s *commandService) getDynamicClients() (dynamic.Interface, discovery.DiscoveryInterface, error) {
	cfg := s.k8sClient.RestConfig()

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create dynamic client: %w", err)
	}

	discClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create discovery client: %w", err)
	}

	return dynClient, discClient, nil
}

// parseMultiDocYAML splits a multi-document YAML string into Unstructured objects
func parseMultiDocYAML(yamlContent string) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(yamlContent)), 4096)
	decSerializer := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	for {
		var rawObj map[string]interface{}
		err := decoder.Decode(&rawObj)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode error: %w", err)
		}
		if rawObj == nil {
			continue
		}

		// Re-encode and decode through the serializer to get proper GVK
		rawJSON, err := json.Marshal(rawObj)
		if err != nil {
			return nil, err
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := decSerializer.Decode(rawJSON, nil, obj)
		if err != nil {
			return nil, fmt.Errorf("decode unstructured: %w", err)
		}
		if gvk != nil {
			obj.SetGroupVersionKind(*gvk)
		}

		objects = append(objects, obj)
	}

	return objects, nil
}
