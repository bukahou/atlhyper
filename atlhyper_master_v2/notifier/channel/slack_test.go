package channel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 占位：不是任何真实系统的地址
const secretPath = "/services/T000/B000/placeholder-not-a-real-token"

func TestSlackNotifier_SendsPayload(t *testing.T) {
	var gotBody slackPayload
	var gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL + secretPath)
	err := notifier.Send(context.Background(), &Message{Subject: "[prod] 高 · shop/deployment/api", Body: "*正文*"})
	if err != nil {
		t.Fatalf("发送应成功: %v", err)
	}
	if gotContentType != "application/json" || gotBody.Text != "[prod] 高 · shop/deployment/api" ||
		len(gotBody.Blocks) != 1 || gotBody.Blocks[0].Text.Text != "*正文*" {
		t.Errorf("请求体不对: contentType=%q body=%+v", gotContentType, gotBody)
	}
}

// Go 的 http.Client 报错会带完整 URL（Post "https://…/services/XXX": …）。
// Webhook 地址本身就是凭据：错误信息里不能有它的任何一段 —— 路径、主机、端口都不行
func TestSlackNotifier_ErrorsNeverContainWebhookURL(t *testing.T) {
	// 起一个服务再关掉：地址合法，但连接一定被拒绝
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedURL := closed.URL
	closed.Close()

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 对端把请求地址原样写回响应体：也不能因此泄漏
		http.Error(w, "bad request to "+r.Host+r.URL.Path, http.StatusInternalServerError)
	}))
	defer failing.Close()

	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:1/elsewhere", http.StatusFound)
	}))
	defer redirecting.Close()

	cases := []struct {
		name    string
		baseURL string
	}{
		{"连接被拒绝", closedURL},
		{"对端返回 500 并回显地址", failing.URL},
		{"对端重定向（不跟随）", redirecting.URL},
		{"主机名解析不了", "http://webhook-host.invalid"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			webhookURL := c.baseURL + secretPath
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := NewSlackNotifier(webhookURL).Send(ctx, &Message{Subject: "s", Body: "b"})
			if err == nil {
				t.Fatal("应当失败")
			}
			message := err.Error()
			host := strings.TrimPrefix(c.baseURL, "http://")
			for _, leaked := range []string{webhookURL, secretPath, "placeholder-not-a-real-token", host, "webhook-host.invalid"} {
				if strings.Contains(message, leaked) {
					t.Errorf("错误信息泄漏了 %q：%s", leaked, message)
				}
			}
			if strings.TrimSpace(message) == "" {
				t.Error("错误信息不应为空：管理员需要知道失败的大致原因")
			}
		})
	}
}

func TestSlackNotifier_NonOKStatusIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid_payload", http.StatusBadRequest)
	}))
	defer server.Close()

	err := NewSlackNotifier(server.URL+secretPath).Send(context.Background(), &Message{Subject: "s", Body: "b"})
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "invalid_payload") {
		t.Errorf("应报出状态码与对端给的原因，实际 %v", err)
	}
}

func TestScrubSecrets(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		secrets []string
		want    string
	}{
		{"替换出现的凭据", "auth failed for pw-123 again pw-123", []string{"pw-123"}, "auth failed for [已隐藏] again [已隐藏]"},
		{"空凭据不参与替换", "plain text", []string{""}, "plain text"},
		{"过短的凭据不替换（否则会把正常文字打碎）", "a b c", []string{"a"}, "a b c"},
		{"长的先替换：不留下短凭据打碎后的残片", "x secret-long-value y", []string{"secret", "secret-long-value"}, "x [已隐藏] y"},
	}
	for _, c := range cases {
		if got := ScrubSecrets(c.text, c.secrets...); got != c.want {
			t.Errorf("%s：得到 %q，期望 %q", c.name, got, c.want)
		}
	}
}
