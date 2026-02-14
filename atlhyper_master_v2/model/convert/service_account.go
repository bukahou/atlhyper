// atlhyper_master_v2/model/convert/service_account.go
// model_v2.ServiceAccount → model.ServiceAccountItem 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// ServiceAccountItem 转换为列表项
func ServiceAccountItem(src *model_v2.ServiceAccount) model.ServiceAccountItem {
	return model.ServiceAccountItem{
		Name:                         src.Name,
		Namespace:                    src.Namespace,
		SecretsCount:                 src.SecretsCount,
		ImagePullSecretsCount:        src.ImagePullSecretsCount,
		AutomountServiceAccountToken: src.AutomountServiceAccountToken,
		CreatedAt:                    src.CreatedAt,
		Age:                          src.Age,
	}
}

// ServiceAccountItems 转换多个 ServiceAccount 为列表项
func ServiceAccountItems(src []model_v2.ServiceAccount) []model.ServiceAccountItem {
	if src == nil {
		return []model.ServiceAccountItem{}
	}
	result := make([]model.ServiceAccountItem, len(src))
	for i := range src {
		result[i] = ServiceAccountItem(&src[i])
	}
	return result
}

// ServiceAccountDetail 转换为详情
func ServiceAccountDetail(src *model_v2.ServiceAccount) model.ServiceAccountDetail {
	return model.ServiceAccountDetail{
		Name:                         src.Name,
		Namespace:                    src.Namespace,
		SecretsCount:                 src.SecretsCount,
		ImagePullSecretsCount:        src.ImagePullSecretsCount,
		AutomountServiceAccountToken: src.AutomountServiceAccountToken,
		SecretNames:                  src.SecretNames,
		ImagePullSecretNames:         src.ImagePullSecretNames,
		CreatedAt:                    src.CreatedAt,
		Age:                          src.Age,
		Labels:                       src.Labels,
		Annotations:                  src.Annotations,
	}
}
