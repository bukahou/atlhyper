package command

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildAPIPath_AllKindMappings(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		kind      string
		namespace string
		resName   string
		expected  string
		wantErr   bool
	}{
		{
			name:      "list pods in namespace",
			command:   "list",
			kind:      "Pod",
			namespace: "default",
			resName:   "",
			expected:  "/api/v1/namespaces/default/pods",
		},
		{
			name:      "list pods all namespaces",
			command:   "list",
			kind:      "Pod",
			namespace: "",
			resName:   "",
			expected:  "/api/v1/pods",
		},
		{
			name:      "get pod by name",
			command:   "get",
			kind:      "Pod",
			namespace: "default",
			resName:   "nginx",
			expected:  "/api/v1/namespaces/default/pods/nginx",
		},
		{
			name:      "list nodes (cluster scope)",
			command:   "list",
			kind:      "Node",
			namespace: "",
			resName:   "",
			expected:  "/api/v1/nodes",
		},
		{
			name:      "get node by name",
			command:   "get",
			kind:      "Node",
			namespace: "",
			resName:   "node1",
			expected:  "/api/v1/nodes/node1",
		},
		{
			name:      "list deployments in namespace",
			command:   "list",
			kind:      "Deployment",
			namespace: "prod",
			resName:   "",
			expected:  "/apis/apps/v1/namespaces/prod/deployments",
		},
		{
			name:      "get deployment by name",
			command:   "get",
			kind:      "Deployment",
			namespace: "prod",
			resName:   "web",
			expected:  "/apis/apps/v1/namespaces/prod/deployments/web",
		},
		{
			name:      "list jobs in namespace",
			command:   "list",
			kind:      "Job",
			namespace: "default",
			resName:   "",
			expected:  "/apis/batch/v1/namespaces/default/jobs",
		},
		{
			name:      "list ingresses in namespace",
			command:   "list",
			kind:      "Ingress",
			namespace: "default",
			resName:   "",
			expected:  "/apis/networking.k8s.io/v1/namespaces/default/ingresses",
		},
		{
			name:      "get_events in namespace",
			command:   "get_events",
			kind:      "",
			namespace: "default",
			resName:   "",
			expected:  "/api/v1/namespaces/default/events",
		},
		{
			name:      "get_events all namespaces",
			command:   "get_events",
			kind:      "",
			namespace: "",
			resName:   "",
			expected:  "/api/v1/events",
		},
		{
			name:      "list statefulsets",
			command:   "list",
			kind:      "StatefulSet",
			namespace: "default",
			resName:   "",
			expected:  "/apis/apps/v1/namespaces/default/statefulsets",
		},
		{
			name:      "list daemonsets",
			command:   "list",
			kind:      "DaemonSet",
			namespace: "kube-system",
			resName:   "",
			expected:  "/apis/apps/v1/namespaces/kube-system/daemonsets",
		},
		{
			name:      "list namespaces (cluster scope)",
			command:   "list",
			kind:      "Namespace",
			namespace: "",
			resName:   "",
			expected:  "/api/v1/namespaces",
		},
		{
			name:      "list persistent volumes (cluster scope)",
			command:   "list",
			kind:      "PersistentVolume",
			namespace: "",
			resName:   "",
			expected:  "/api/v1/persistentvolumes",
		},
		{
			name:      "describe pod",
			command:   "describe",
			kind:      "Pod",
			namespace: "default",
			resName:   "nginx",
			expected:  "/api/v1/namespaces/default/pods/nginx",
		},
		{
			name:      "list cronjobs",
			command:   "list",
			kind:      "CronJob",
			namespace: "default",
			resName:   "",
			expected:  "/apis/batch/v1/namespaces/default/cronjobs",
		},
		{
			name:      "list network policies",
			command:   "list",
			kind:      "NetworkPolicy",
			namespace: "default",
			resName:   "",
			expected:  "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies",
		},
		{
			name:      "list HPA",
			command:   "list",
			kind:      "HPA",
			namespace: "default",
			resName:   "",
			expected:  "/apis/autoscaling/v2/namespaces/default/horizontalpodautoscalers",
		},
		// Error cases
		{
			name:      "get pod without name",
			command:   "get",
			kind:      "Pod",
			namespace: "ns",
			resName:   "",
			wantErr:   true,
		},
		{
			name:      "get namespaced resource without namespace",
			command:   "get",
			kind:      "Pod",
			namespace: "",
			resName:   "nginx",
			wantErr:   true,
		},
		{
			name:      "unsupported kind",
			command:   "list",
			kind:      "UnknownKind",
			namespace: "",
			resName:   "",
			wantErr:   true,
		},
		{
			name:      "unsupported command",
			command:   "unknown_command",
			kind:      "Pod",
			namespace: "",
			resName:   "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildAPIPath(tt.command, tt.kind, tt.namespace, tt.resName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("buildAPIPath(%q, %q, %q, %q) expected error, got path %q",
						tt.command, tt.kind, tt.namespace, tt.resName, got)
				}
				return
			}

			if err != nil {
				t.Errorf("buildAPIPath(%q, %q, %q, %q) unexpected error: %v",
					tt.command, tt.kind, tt.namespace, tt.resName, err)
				return
			}

			if got != tt.expected {
				t.Errorf("buildAPIPath(%q, %q, %q, %q) = %q, want %q",
					tt.command, tt.kind, tt.namespace, tt.resName, got, tt.expected)
			}
		})
	}
}

func TestStripManagedFields_RemoveFromObject(t *testing.T) {
	input := map[string]interface{}{
		"kind":       "Pod",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name":      "nginx",
			"namespace": "default",
			"managedFields": []interface{}{
				map[string]interface{}{
					"manager":   "kubectl",
					"operation": "Apply",
				},
			},
		},
		"spec": map[string]interface{}{
			"containers": []interface{}{},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	result := stripManagedFields(data)

	var output map[string]interface{}
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	meta, ok := output["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata not found in output")
	}

	if _, hasMF := meta["managedFields"]; hasMF {
		t.Error("managedFields should have been removed from metadata")
	}

	// Verify other metadata fields preserved
	if name, _ := meta["name"].(string); name != "nginx" {
		t.Errorf("metadata.name should be preserved, got %q", name)
	}
	if ns, _ := meta["namespace"].(string); ns != "default" {
		t.Errorf("metadata.namespace should be preserved, got %q", ns)
	}

	// Verify spec preserved
	if _, hasSpec := output["spec"]; !hasSpec {
		t.Error("spec should be preserved in output")
	}
}

func TestStripManagedFields_RemoveFromList(t *testing.T) {
	input := map[string]interface{}{
		"kind":       "PodList",
		"apiVersion": "v1",
		"items": []interface{}{
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "pod-1",
					"managedFields": []interface{}{
						map[string]interface{}{"manager": "kubectl"},
					},
				},
			},
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "pod-2",
					"managedFields": []interface{}{
						map[string]interface{}{"manager": "kube-controller"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	result := stripManagedFields(data)

	var output map[string]interface{}
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	items, ok := output["items"].([]interface{})
	if !ok {
		t.Fatal("items not found in output")
	}

	for i, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("item %d is not a map", i)
		}
		meta, ok := m["metadata"].(map[string]interface{})
		if !ok {
			t.Fatalf("item %d metadata not found", i)
		}
		if _, hasMF := meta["managedFields"]; hasMF {
			t.Errorf("item %d still has managedFields", i)
		}
		// Verify name preserved
		if _, hasName := meta["name"]; !hasName {
			t.Errorf("item %d metadata.name should be preserved", i)
		}
	}
}

func TestStripManagedFields_InvalidJSON(t *testing.T) {
	input := []byte(`{not valid json}`)
	result := stripManagedFields(input)

	if string(result) != string(input) {
		t.Errorf("expected invalid JSON returned as-is, got %q", string(result))
	}
}

func TestStripManagedFields_NoManagedFields(t *testing.T) {
	input := `{"metadata":{"name":"test"},"spec":{}}`
	result := stripManagedFields([]byte(input))

	var output map[string]interface{}
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	meta, ok := output["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata not found")
	}
	if name, _ := meta["name"].(string); name != "test" {
		t.Errorf("expected name 'test', got %q", name)
	}
}

func TestBuildEventFieldSelector(t *testing.T) {
	tests := []struct {
		name         string
		involvedKind string
		involvedName string
		expected     string
	}{
		{
			name:         "both kind and name",
			involvedKind: "Pod",
			involvedName: "nginx",
			expected:     "involvedObject.kind=Pod,involvedObject.name=nginx",
		},
		{
			name:         "kind only",
			involvedKind: "Pod",
			involvedName: "",
			expected:     "involvedObject.kind=Pod",
		},
		{
			name:         "name only",
			involvedKind: "",
			involvedName: "nginx",
			expected:     "involvedObject.name=nginx",
		},
		{
			name:         "both empty",
			involvedKind: "",
			involvedName: "",
			expected:     "",
		},
		{
			name:         "deployment kind",
			involvedKind: "Deployment",
			involvedName: "web-app",
			expected:     "involvedObject.kind=Deployment,involvedObject.name=web-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildEventFieldSelector(tt.involvedKind, tt.involvedName)
			if got != tt.expected {
				t.Errorf("buildEventFieldSelector(%q, %q) = %q, want %q",
					tt.involvedKind, tt.involvedName, got, tt.expected)
			}
		})
	}
}

func TestBuildAPIPath_NodeWithNamespace(t *testing.T) {
	// Node is cluster-scoped, so namespace is ignored for list
	got, err := buildAPIPath("list", "Node", "default", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ClusterScope=true means namespace is ignored
	expected := "/api/v1/nodes"
	if got != expected {
		t.Errorf("expected %q for cluster-scoped resource with namespace, got %q", expected, got)
	}
}

func TestBuildAPIPath_Services(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		namespace string
		resName   string
		expected  string
	}{
		{
			name:      "list services all namespaces",
			command:   "list",
			namespace: "",
			resName:   "",
			expected:  "/api/v1/services",
		},
		{
			name:      "list services in namespace",
			command:   "list",
			namespace: "kube-system",
			resName:   "",
			expected:  "/api/v1/namespaces/kube-system/services",
		},
		{
			name:      "get service by name",
			command:   "get",
			namespace: "default",
			resName:   "kubernetes",
			expected:  "/api/v1/namespaces/default/services/kubernetes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildAPIPath(tt.command, "Service", tt.namespace, tt.resName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildAPIPath_DescribeRequiresName(t *testing.T) {
	_, err := buildAPIPath("describe", "Deployment", "default", "")
	if err == nil {
		t.Error("expected error when describe has no name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("expected 'name is required' in error, got %q", err.Error())
	}
}

func TestBuildAPIPath_DescribeNamespacedWithoutNamespace(t *testing.T) {
	_, err := buildAPIPath("describe", "Pod", "", "nginx")
	if err == nil {
		t.Error("expected error when describe namespaced resource without namespace")
	}
	if !strings.Contains(err.Error(), "namespace is required") {
		t.Errorf("expected 'namespace is required' in error, got %q", err.Error())
	}
}
