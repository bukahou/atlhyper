// atlhyper_master_v2/gateway/middleware/audit.go
// 审计中间件 - 记录敏感操作到审计日志
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// AuditConfig 审计配置
type AuditConfig struct {
	Action   string // 操作类型：login, create, delete, update, execute
	Resource string // 资源类型：user, command, notify, cluster
}

// AuditRepository 审计日志仓库接口（避免循环依赖）
type AuditRepository interface {
	Create(ctx context.Context, log *database.AuditLog) error
}

// auditResponseWriter 包装 ResponseWriter 以获取状态码
type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *auditResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush 实现 http.Flusher 接口（SSE 流式响应需要）
func (rw *auditResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Audit 审计中间件
// 记录敏感操作到审计日志
func Audit(repo AuditRepository, config AuditConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 读取请求体（需要复制以便后续处理）
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// 包装 ResponseWriter
			rw := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// 处理请求
			next.ServeHTTP(rw, r)

			// 异步记录审计日志
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// 获取用户信息（可能从 context 中获取）
				userID, _ := GetUserID(r.Context())
				username, _ := GetUsername(r.Context())
				role, _ := GetRole(r.Context())

				// 获取客户端 IP
				ip := r.RemoteAddr
				if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
					ip = strings.Split(xForwardedFor, ",")[0]
				}

				// 脱敏请求体
				sanitizedBody := sanitizeRequestBody(string(bodyBytes))

				log := &database.AuditLog{
					Timestamp:   start,
					UserID:      userID,
					Username:    username,
					Role:        role,
					Source:      "web",
					Action:      config.Action,
					Resource:    config.Resource,
					Method:      r.Method,
					Path:        r.URL.Path, // 只记路径：查询串可能带凭据
					RequestBody: sanitizedBody,
					StatusCode:  rw.statusCode,
					Success:     rw.statusCode >= 200 && rw.statusCode < 300,
					IP:          ip,
					UserAgent:   r.Header.Get("User-Agent"),
					DurationMs:  time.Since(start).Milliseconds(),
				}

				// 如果失败，记录错误信息
				if !log.Success {
					log.ErrorMessage = http.StatusText(rw.statusCode)
				}

				repo.Create(ctx, log)
			}()
		}
	}
}

// sensitiveKeyParts 键名（小写、去掉 _ 与 -）含这些片段即视为凭据
var sensitiveKeyParts = []string{
	"password", "passwd", "secret", "token", "apikey", "accesskey", "privatekey",
	"credential", "authorization", "cookie", "webhook",
}

const (
	redactedMark    = "***"
	maxAuditBodyLen = 500
)

// sanitizeRequestBody 脱敏请求体后写入审计：按 JSON 解析，任意层级的凭据类键，其字符串值
// （或整个对象 / 数组）替换为 ***；数字与布尔不是凭据，原样保留（token 用量、上限等）。
// 不是 JSON 的请求体不记录内容（失败即关）。
func sanitizeRequestBody(body string) string {
	if len(body) == 0 {
		return ""
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return "[非 JSON 请求体，未记录内容]"
	}
	out, err := json.Marshal(redactCredentials(parsed))
	if err != nil {
		return "[请求体无法记录]"
	}
	if len(out) > maxAuditBodyLen {
		return string(out[:maxAuditBodyLen]) + "..."
	}
	return string(out)
}

// redactCredentials 递归替换凭据类键的值
func redactCredentials(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if isCredentialKey(k) {
				switch val.(type) {
				case string, map[string]interface{}, []interface{}:
					t[k] = redactedMark
					continue
				}
			}
			t[k] = redactCredentials(val)
		}
	case []interface{}:
		for i := range t {
			t[i] = redactCredentials(t[i])
		}
	}
	return v
}

func isCredentialKey(key string) bool {
	k := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
	for _, part := range sensitiveKeyParts {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}

// AuditAction 审计操作常量
const (
	AuditActionLogin   = "login"
	AuditActionCreate  = "create"
	AuditActionUpdate  = "update"
	AuditActionDelete  = "delete"
	AuditActionExecute = "execute"
)

// AuditResource 审计资源常量
const (
	AuditResourceUser    = "user"
	AuditResourceCommand = "command"
	AuditResourceNotify  = "notify"
	AuditResourceCluster = "cluster"
)
