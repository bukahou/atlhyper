package command

import (
	"context"
	"errors"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// testMapper 认识几种常见资源：命名空间级与集群级各几种
func testMapper() meta.RESTMapper {
	m := meta.NewDefaultRESTMapper(nil)
	namespaced := []schema.GroupVersionKind{
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		{Group: "", Version: "v1", Kind: "Service"},
		{Group: "", Version: "v1", Kind: "ConfigMap"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
	}
	cluster := []schema.GroupVersionKind{
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
		{Group: "", Version: "v1", Kind: "Namespace"},
	}
	for _, gvk := range namespaced {
		m.Add(gvk, meta.RESTScopeNamespace)
	}
	for _, gvk := range cluster {
		m.Add(gvk, meta.RESTScopeRoot)
	}
	return m
}

func mustParse(t *testing.T, manifests string) []*unstructured.Unstructured {
	t.Helper()
	objects, err := parseMultiDocYAML(manifests)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return objects
}

const deployAndService = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: shop
---
apiVersion: v1
kind: Service
metadata:
  name: web
  namespace: shop
`

func TestPlanApply_AcceptsAllowedNamespacedKinds(t *testing.T) {
	plan, err := planApply(mustParse(t, deployAndService), testMapper())
	if err != nil {
		t.Fatalf("planApply: %v", err)
	}
	if len(plan) != 2 {
		t.Fatalf("plan 有 %d 项，want 2", len(plan))
	}
	if plan[0].namespace != "shop" || plan[0].resource.Resource != "deployments" {
		t.Errorf("第一项 = %+v", plan[0])
	}
}

func TestPlanApply_EmptyNamespaceMeansDefault(t *testing.T) {
	plan, err := planApply(mustParse(t, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: c\n"), testMapper())
	if err != nil {
		t.Fatalf("planApply: %v", err)
	}
	if plan[0].namespace != "default" {
		t.Errorf("namespace = %q, want default", plan[0].namespace)
	}
}

// 整批只要有一项不合格就整批拒绝：一个都不应用
func TestPlanApply_RejectsWholeBatch(t *testing.T) {
	cases := []struct {
		name     string
		extra    string
		wantText string
	}{
		{"集群级资源", "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRoleBinding\nmetadata:\n  name: x\n", "ClusterRoleBinding"},
		{"命名空间本身", "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: x\n", "Namespace"},
		{"权限绑定", "apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: x\n  namespace: shop\n", "RoleBinding"},
		{"不在名单的工作负载", "apiVersion: apps/v1\nkind: DaemonSet\nmetadata:\n  name: x\n  namespace: shop\n", "DaemonSet"},
		{"系统命名空间", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n  namespace: kube-system\n", "kube-system"},
		{"集群不认识的类型", "apiVersion: example.com/v1\nkind: Widget\nmetadata:\n  name: x\n  namespace: shop\n", "Widget"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := planApply(mustParse(t, deployAndService+"---\n"+c.extra), testMapper())
			if err == nil {
				t.Fatal("期望整批被拒绝")
			}
			if !strings.Contains(err.Error(), c.wantText) {
				t.Errorf("错误应点名 %q，实际 %v", c.wantText, err)
			}
		})
	}
}

func TestPlanApply_RejectsEmptyBatch(t *testing.T) {
	if _, err := planApply(nil, testMapper()); err == nil {
		t.Fatal("空清单应被拒绝")
	}
}

// fakeApplier 记录调用顺序；failOn 里的「名字 + 是否预演」返回错误
type fakeApplier struct {
	calls  []string
	failOn map[string]bool
}

func (f *fakeApplier) Apply(_ context.Context, item planItem, dryRun bool) error {
	key := item.object.GetName()
	if dryRun {
		key = "dry:" + key
	}
	f.calls = append(f.calls, key)
	if f.failOn[key] {
		return errors.New("rejected by api server")
	}
	return nil
}

func twoItemPlan(t *testing.T) []planItem {
	plan, err := planApply(mustParse(t, strings.ReplaceAll(deployAndService, "name: web\n  namespace: shop\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: web", "name: web\n  namespace: shop\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: web-svc")), testMapper())
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestRunApply_DryRunsEverythingBeforeApplying(t *testing.T) {
	applier := &fakeApplier{}
	changed, err := runApply(context.Background(), twoItemPlan(t), applier)
	if err != nil {
		t.Fatalf("runApply: %v", err)
	}
	want := []string{"dry:web", "dry:web-svc", "web", "web-svc"}
	if strings.Join(applier.calls, ",") != strings.Join(want, ",") {
		t.Errorf("调用顺序 = %v, want %v", applier.calls, want)
	}
	if changed != 2 {
		t.Errorf("changed = %d, want 2", changed)
	}
}

func TestRunApply_DryRunFailureAppliesNothing(t *testing.T) {
	applier := &fakeApplier{failOn: map[string]bool{"dry:web-svc": true}}
	_, err := runApply(context.Background(), twoItemPlan(t), applier)
	if err == nil {
		t.Fatal("预演失败应报错")
	}
	for _, call := range applier.calls {
		if !strings.HasPrefix(call, "dry:") {
			t.Errorf("预演失败后不应真正应用，实际调用了 %s", call)
		}
	}
}

// 真正应用时有一项失败：整条指令失败，错误里带上已应用的数量
func TestRunApply_PartialFailureIsFailure(t *testing.T) {
	applier := &fakeApplier{failOn: map[string]bool{"web-svc": true}}
	changed, err := runApply(context.Background(), twoItemPlan(t), applier)
	if err == nil {
		t.Fatal("部分失败应报错")
	}
	if changed != 1 {
		t.Errorf("changed = %d, want 1", changed)
	}
	if !strings.Contains(err.Error(), "1/2") {
		t.Errorf("错误应带上已应用数量 1/2，实际 %v", err)
	}
}
