package command

import (
	"strings"
	"testing"
)

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		target    string
		wantErr   bool
	}{
		{"都为空（集群级列表）", "", "", false},
		{"普通 Pod", "default", "nginx-7c5ddbdf54-x8k2p", false},
		{"带点的名字（节点、ConfigMap）", "", "node-1.example.internal", false},
		{"kube-root-ca.crt", "kube-system", "kube-root-ca.crt", false},
		{"数字开头", "default", "1password", false},
		{"名字长度上限 253", "default", strings.Repeat("a", 253), false},
		{"命名空间长度上限 63", strings.Repeat("a", 63), "x", false},

		{"名字带斜杠", "", "kube-system/secrets", true},
		{"名字是上级目录", "default", "..", true},
		{"名字带上级目录片段", "default", "a/../b", true},
		{"名字带百分号编码", "default", "a%2fb", true},
		{"名字带问号", "default", "a?watch=true", true},
		{"名字带空格", "default", "a b", true},
		{"名字带大写", "default", "Nginx", true},
		{"名字以点开头", "default", ".hidden", true},
		{"名字以横线结尾", "default", "nginx-", true},
		{"名字带连续的点", "default", "a..b", true},
		{"名字带换行", "default", "a\nb", true},
		{"名字超长", "default", strings.Repeat("a", 254), true},
		{"命名空间带斜杠", "default/secrets", "x", true},
		{"命名空间带点", "kube.system", "x", true},
		{"命名空间超长", strings.Repeat("a", 64), "x", true},
		{"命名空间带大写", "Default", "x", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTarget(tt.namespace, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTarget(%q, %q) error = %v, wantErr %v", tt.namespace, tt.target, err, tt.wantErr)
			}
		})
	}
}
