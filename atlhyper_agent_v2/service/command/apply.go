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
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// ApplyManifestsResult holds the result of applying manifests
type ApplyManifestsResult struct {
	ResourceTotal   int    `json:"resourceTotal"`
	ResourceChanged int    `json:"resourceChanged"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
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

	result := ApplyManifestsResult{
		ResourceTotal: len(objects),
	}

	var errs []string
	for _, obj := range objects {
		gvk := obj.GroupVersionKind()
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s/%s: %v", gvk.Kind, obj.GetName(), err))
			continue
		}

		// Determine if namespaced or cluster-scoped
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ns := obj.GetNamespace()
			if ns == "" {
				ns = "default"
			}
			dr = dynClient.Resource(mapping.Resource).Namespace(ns)
		} else {
			dr = dynClient.Resource(mapping.Resource)
		}

		// Marshal to JSON for SSA
		data, err := json.Marshal(obj.Object)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s/%s: marshal error: %v", gvk.Kind, obj.GetName(), err))
			continue
		}

		// Server-Side Apply
		_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "atlhyper-agent",
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s/%s: %v", gvk.Kind, obj.GetName(), err))
			continue
		}

		result.ResourceChanged++
	}

	if len(errs) > 0 {
		result.ErrorMessage = strings.Join(errs, "; ")
	}

	out, _ := json.Marshal(result)
	return string(out), nil
}

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
