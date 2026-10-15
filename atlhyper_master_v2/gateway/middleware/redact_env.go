// atlhyper_master_v2/gateway/middleware/redact_env.go
// 资源详情的 env 字面值裁剪（装饰器）
//
// 各资源详情的容器结构不统一（Pod 为强类型，工作负载 template 为 interface{}），
// 因此在 JSON 层统一处理：任意层级的 envs[] 元素去掉 value，标记 valueHidden；
// name 与 valueFrom（如 secret:xxx）保留。新增资源类型自动覆盖。
package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

// RedactEnvValues 主体缺少 perm 时，剥掉响应中所有 env 字面值
func RedactEnvValues(perm Permission, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if Allowed(GetPrincipal(r.Context()).Level, perm) {
			next(w, r)
			return
		}

		buf := &bufferedResponse{header: http.Header{}, status: http.StatusOK}
		next(buf, r)

		body := buf.body.Bytes()
		if buf.status == http.StatusOK {
			var doc any
			if err := json.Unmarshal(body, &doc); err != nil {
				// 失败即关：无法确认内容时不吐出原文
				http.Error(w, `{"error": "响应处理失败"}`, http.StatusInternalServerError)
				return
			}
			stripEnvValues(doc)
			redacted, err := json.Marshal(doc)
			if err != nil {
				http.Error(w, `{"error": "响应处理失败"}`, http.StatusInternalServerError)
				return
			}
			body = redacted
		}

		for k, v := range buf.header {
			w.Header()[k] = v
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(buf.status)
		w.Write(body)
	}
}

// stripEnvValues 递归处理任意层级的 "envs" 数组
func stripEnvValues(node any) {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "envs" {
				if items, ok := child.([]any); ok {
					for _, item := range items {
						if env, ok := item.(map[string]any); ok {
							if val, has := env["value"]; has && val != "" {
								delete(env, "value")
								env["valueHidden"] = true
							}
						}
					}
				}
				continue
			}
			stripEnvValues(child)
		}
	case []any:
		for _, child := range v {
			stripEnvValues(child)
		}
	}
}

// bufferedResponse 暂存下游响应，供改写后再写出
type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedResponse) Header() http.Header         { return b.header }
func (b *bufferedResponse) Write(p []byte) (int, error) { return b.body.Write(p) }
func (b *bufferedResponse) WriteHeader(status int)      { b.status = status }
