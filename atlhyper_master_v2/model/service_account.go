// atlhyper_master_v2/model/service_account.go
// ServiceAccount Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// ServiceAccountItem ServiceAccount 列表项
type ServiceAccountItem struct {
	Name                         string `json:"name"`
	Namespace                    string `json:"namespace"`
	SecretsCount                 int    `json:"secretsCount"`
	ImagePullSecretsCount        int    `json:"imagePullSecretsCount"`
	AutomountServiceAccountToken *bool  `json:"automountServiceAccountToken,omitempty"`
	CreatedAt                    string `json:"createdAt"`
	Age                          string `json:"age"`
}

// ServiceAccountDetail ServiceAccount 详情
type ServiceAccountDetail struct {
	Name                         string   `json:"name"`
	Namespace                    string   `json:"namespace"`
	SecretsCount                 int      `json:"secretsCount"`
	ImagePullSecretsCount        int      `json:"imagePullSecretsCount"`
	AutomountServiceAccountToken *bool    `json:"automountServiceAccountToken,omitempty"`
	SecretNames                  []string `json:"secretNames,omitempty"`
	ImagePullSecretNames         []string `json:"imagePullSecretNames,omitempty"`
	CreatedAt                    string   `json:"createdAt"`
	Age                          string   `json:"age"`

	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
