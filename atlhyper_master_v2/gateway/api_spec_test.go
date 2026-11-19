package gateway

import (
	"strings"
	"testing"

	"AtlHyper/api/contract"
)

// 规格里的每个接口都已在路由表注册，且规格声明的权限（x-permission）与路由表一致：
// 权限只在路由表定义一处，规格跟着它走，⛔ 两边各写各的。
func TestAPISpec_OperationsMatchRoutes(t *testing.T) {
	doc, err := contract.Spec()
	if err != nil {
		t.Fatal(err)
	}
	r, _ := testRouterWithSpecs(t)
	registered := map[string]routeSpec{}
	for _, s := range r.specs {
		registered[s.method+" "+s.path] = s
	}

	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			key := strings.ToUpper(method) + " " + path
			route, ok := registered[key]
			if !ok {
				t.Errorf("规格里有 %s，路由表里没有", key)
				continue
			}
			declared, _ := op.Extensions["x-permission"].(string)
			if declared == "" {
				t.Errorf("%s 在规格里没有声明 x-permission", key)
				continue
			}
			if declared != string(route.perm) {
				t.Errorf("%s 的权限：规格 %q，路由表 %q", key, declared, route.perm)
			}
		}
	}
}
