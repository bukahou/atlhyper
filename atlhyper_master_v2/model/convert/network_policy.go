// atlhyper_master_v2/model/convert/network_policy.go
// model_v2.NetworkPolicy → model.NetworkPolicyItem 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// NetworkPolicyItem 转换为列表项
func NetworkPolicyItem(src *model_v2.NetworkPolicy) model.NetworkPolicyItem {
	return model.NetworkPolicyItem{
		Name:             src.Name,
		Namespace:        src.Namespace,
		PodSelector:      src.PodSelector,
		PolicyTypes:      src.PolicyTypes,
		IngressRuleCount: src.IngressRuleCount,
		EgressRuleCount:  src.EgressRuleCount,
		CreatedAt:        src.CreatedAt,
		Age:              src.Age,
	}
}

// NetworkPolicyItems 转换多个 NetworkPolicy 为列表项
func NetworkPolicyItems(src []model_v2.NetworkPolicy) []model.NetworkPolicyItem {
	if src == nil {
		return []model.NetworkPolicyItem{}
	}
	result := make([]model.NetworkPolicyItem, len(src))
	for i := range src {
		result[i] = NetworkPolicyItem(&src[i])
	}
	return result
}

// NetworkPolicyDetail 转换为详情
func NetworkPolicyDetail(src *model_v2.NetworkPolicy) model.NetworkPolicyDetail {
	detail := model.NetworkPolicyDetail{
		Name:             src.Name,
		Namespace:        src.Namespace,
		PodSelector:      src.PodSelector,
		PolicyTypes:      src.PolicyTypes,
		IngressRuleCount: src.IngressRuleCount,
		EgressRuleCount:  src.EgressRuleCount,
		CreatedAt:        src.CreatedAt,
		Age:              src.Age,
		Labels:           src.Labels,
		Annotations:      src.Annotations,
	}

	if len(src.IngressRules) > 0 {
		detail.IngressRules = src.IngressRules
	}
	if len(src.EgressRules) > 0 {
		detail.EgressRules = src.EgressRules
	}

	return detail
}
