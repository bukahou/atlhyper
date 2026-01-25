// internal/readonly/ingress/convert_ingressclass.go
package ingress

import (
	"context"

	"AtlHyper/atlhyper_agent/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fetchIngressClassControllers 拉取 IngressClass，返回 map[className]controller
// 失败时返回空表，不阻断主流程。
func fetchIngressClassControllers(ctx context.Context) (map[string]string, error) {
	cs := utils.GetCoreClient()
	classes, err := cs.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
	if err != nil || classes == nil {
		return map[string]string{}, err
	}
	out := make(map[string]string, len(classes.Items))
	for i := range classes.Items {
		ic := &classes.Items[i]
		out[ic.Name] = ic.Spec.Controller
	}
	return out, nil
}
