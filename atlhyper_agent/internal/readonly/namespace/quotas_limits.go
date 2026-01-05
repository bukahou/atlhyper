// internal/readonly/namespace/quotas_limits.go
package namespace

import (
	"context"

	modelns "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// indexQuotasByNS —— 全量拉取 ResourceQuota，映射为模型并返回是否配额超限集合
func indexQuotasByNS(ctx context.Context, cs *kubernetes.Clientset) (map[string][]modelns.ResourceQuota, map[string]bool, error) {
	out := map[string][]modelns.ResourceQuota{}
	exceededNS := map[string]bool{}

	rqs, err := cs.CoreV1().ResourceQuotas(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil || rqs == nil {
		return out, exceededNS, err
	}
	for i := range rqs.Items {
		rq := &rqs.Items[i]
		hard := qtyMapToStrMap(rq.Spec.Hard)
		used := qtyMapToStrMap(rq.Status.Used)

		item := modelns.ResourceQuota{
			Name:   rq.Name,
			Scopes: scopeStrings(rq.Spec.Scopes),
			Hard:   hard,
			Used:   used,
		}
		out[rq.Namespace] = append(out[rq.Namespace], item)

		// 超限判定（任一资源 used > hard 即认为超限）
		if quotaExceeded(rq.Status.Used, rq.Spec.Hard) {
			exceededNS[rq.Namespace] = true
		}
	}
	return out, exceededNS, nil
}

func scopeStrings(scopes []corev1.ResourceQuotaScope) []string {
	if len(scopes) == 0 {
		return nil
	}
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		out = append(out, string(s))
	}
	return out
}

func qtyMapToStrMap(m corev1.ResourceList) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[string(k)] = v.String()
	}
	return out
}

func quotaExceeded(used, hard corev1.ResourceList) bool {
	for name, hv := range hard {
		if uv, ok := used[name]; ok {
			if cmp := uv.Cmp(hv); cmp > 0 {
				return true
			}
		}
	}
	return false
}

// indexLimitRangesByNS —— 全量拉取 LimitRange，映射为模型
func indexLimitRangesByNS(ctx context.Context, cs *kubernetes.Clientset) (map[string][]modelns.LimitRange, error) {
	out := map[string][]modelns.LimitRange{}

	lrs, err := cs.CoreV1().LimitRanges(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil || lrs == nil {
		return out, err
	}
	for i := range lrs.Items {
		lr := &lrs.Items[i]
		items := make([]modelns.LimitRangeItem, 0, len(lr.Spec.Limits))
		for _, it := range lr.Spec.Limits {
			items = append(items, modelns.LimitRangeItem{
				Type:                 string(it.Type),
				Max:                  qtyMapToStrMap(it.Max),
				Min:                  qtyMapToStrMap(it.Min),
				Default:              qtyMapToStrMap(it.Default),
				DefaultRequest:       qtyMapToStrMap(it.DefaultRequest),
				MaxLimitRequestRatio: qtyMapToStrMap(it.MaxLimitRequestRatio),
			})
		}
		out[lr.Namespace] = append(out[lr.Namespace], modelns.LimitRange{
			Name:  lr.Name,
			Items: items,
		})
	}
	return out, nil
}
