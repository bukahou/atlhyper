// atlhyper_master_v2/gateway/middleware/audit.go
// 审计中间件 - 记录敏感操作到审计日志
package middleware

import (
	"bytes"
	"context"
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

// sanitizeRequestBody 脱敏请求体
// 移除敏感字段如 password
func sanitizeRequestBody(body string) string {
	if len(body) == 0 {
		return ""
	}

	// 简单脱敏：替换 password 字段
	// 可以使用更复杂的 JSON 解析和替换，但这里保持简单
	if strings.Contains(body, "password") {
		// 简单处理：截断包含密码的请求体
		body = strings.ReplaceAll(body, `"password"`, `"password":"***"`)
	}

	// 限制长度
	if len(body) > 500 {
		body = body[:500] + "..."
	}

	return body
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
