package middleware

import (
	"encoding/json"
	"strings"
	"testing"
)

func sanitizedMap(t *testing.T, body string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(sanitizeRequestBody(body)), &m); err != nil {
		t.Fatalf("脱敏结果不是 JSON：%v", err)
	}
	return m
}

// 凭据类键（任意层级、任意大小写与写法）的字符串值一律打码；其余字段原样留作审计证据
func TestSanitizeRequestBody_RedactsCredentialsAtAnyDepth(t *testing.T) {
	cases := []struct {
		name, body string
		leaked     []string // 不得出现在结果里的值
		kept       []string // 必须保留的值
	}{
		{"登录", `{"username":"alice","password":"p-1"}`, []string{"p-1"}, []string{"alice"}},
		{"SMTP 密码（嵌套、驼峰）", `{"config":{"smtpHost":"smtp.example.com","smtpUser":"bot","smtpPassword":"p-2"}}`,
			[]string{"p-2"}, []string{"smtp.example.com", "bot"}},
		{"Slack webhook", `{"enabled":true,"config":{"webhookUrl":"https://hooks.slack.com/services/T/B/x-3"}}`,
			[]string{"x-3"}, nil},
		{"钉钉 webhook（下划线）", `{"config":{"webhook_url":"https://oapi.example.com/robot/send?access_token=x-4"}}`,
			[]string{"x-4"}, nil},
		{"AI Key 两种写法", `{"provider":"openai","api_key":"k-5","apiKey":"k-6","baseUrl":"https://api.example.com"}`,
			[]string{"k-5", "k-6"}, []string{"openai", "https://api.example.com"}},
		{"数组里的 token", `{"items":[{"name":"a","accessToken":"t-7"}]}`, []string{"t-7"}, []string{"a"}},
		{"同名字段出现两次", `{"a":{"password":"p-8"},"b":{"password":"p-9"}}`, []string{"p-8", "p-9"}, nil},
		{"凭据键下的对象整个打码", `{"secrets":{"DB":"s-10"}}`, []string{"s-10"}, nil},
	}
	for _, c := range cases {
		got := sanitizeRequestBody(c.body)
		for _, v := range c.leaked {
			if strings.Contains(got, v) {
				t.Errorf("%s：凭据 %q 泄露到审计里：%s", c.name, v, got)
			}
		}
		for _, v := range c.kept {
			if !strings.Contains(got, v) {
				t.Errorf("%s：非凭据字段 %q 被误删：%s", c.name, v, got)
			}
		}
	}
}

// 数字与布尔不是凭据：token 用量 / 上限、automountServiceAccountToken 等照常记录
func TestSanitizeRequestBody_KeepsNumbersAndBools(t *testing.T) {
	m := sanitizedMap(t, `{"dailyInputTokenLimit":1000,"monthlyOutputTokensUsed":20,"automountServiceAccountToken":false}`)
	if m["dailyInputTokenLimit"] != float64(1000) || m["monthlyOutputTokensUsed"] != float64(20) || m["automountServiceAccountToken"] != false {
		t.Errorf("数字 / 布尔不应打码：%v", m)
	}
}

func TestSanitizeRequestBody_NonJSONNotRecorded(t *testing.T) {
	if got := sanitizeRequestBody("username=a&password=p-11"); strings.Contains(got, "p-11") || got == "" {
		t.Errorf("非 JSON 请求体不记录内容，got %q", got)
	}
	if got := sanitizeRequestBody(""); got != "" {
		t.Errorf("空请求体 = %q", got)
	}
}

func TestSanitizeRequestBody_Truncates(t *testing.T) {
	got := sanitizeRequestBody(`{"note":"` + strings.Repeat("x", 1000) + `"}`)
	if len(got) > 503 || !strings.HasSuffix(got, "...") {
		t.Errorf("超长请求体应截断到 500 字节：len=%d", len(got))
	}
}
