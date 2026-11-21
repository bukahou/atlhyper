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
	"AtlHyper/atlhyper_master_v2/gateway/clientip"
	"AtlHyper/common/logger"
)

var auditLog = logger.Module("Audit")

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

// Unwrap 让 http.ResponseController 拿到底层连接（流式响应逐块续期写超时）
func (rw *auditResponseWriter) Unwrap() http.ResponseWriter { return rw.ResponseWriter }

// Flush 实现 http.Flusher 接口（SSE 流式响应需要）
func (rw *auditResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// maxAuditCaptureBytes 审计最多留存处理函数读到的这么多字节用于脱敏记录；超过则不记内容
const maxAuditCaptureBytes = 64 << 10

// Audit 审计中间件（客户端地址取 TCP 对端；部署配置了可信头时用 AuditWithClientIP）
func Audit(repo AuditRepository, config AuditConfig) func(http.HandlerFunc) http.HandlerFunc {
	return AuditWithClientIP(repo, config, "")
}

// AuditWithClientIP 审计中间件：记录敏感操作到审计日志。
//
// 请求体不预先读取：只留存处理函数真正读到的部分。权限检查没通过时处理函数不读请求体，
// 审计也就不记 —— 不替未鉴权的请求把请求体读进内存。clientIPHeader 见 clientip.FromRequest。
func AuditWithClientIP(repo AuditRepository, config AuditConfig, clientIPHeader string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			var captured *capturingBody
			if r.Body != nil {
				captured = &capturingBody{ReadCloser: r.Body, limit: maxAuditCaptureBytes}
				r.Body = captured
			}

			// 包装 ResponseWriter
			rw := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// 处理请求
			next.ServeHTTP(rw, r)

			var sanitizedBody string
			if captured != nil {
				if captured.overflow {
					sanitizedBody = "[请求体过大，未记录内容]"
				} else {
					sanitizedBody = sanitizeRequestBody(captured.buf.String())
				}
			}
			ip := clientip.FromRequest(r, clientIPHeader)

			// 异步记录审计日志
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				userID, _ := GetUserID(r.Context())
				username, _ := GetUsername(r.Context())
				role, _ := GetRole(r.Context())

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

				if err := repo.Create(ctx, log); err != nil {
					auditLog.Warn("写审计日志失败", "path", log.Path, "err", err)
				}
			}()
		}
	}
}

// capturingBody 处理函数读请求体时顺带留一份（最多 limit 字节），超过即只标记 overflow
type capturingBody struct {
	io.ReadCloser
	buf      bytes.Buffer
	limit    int
	overflow bool
}

func (c *capturingBody) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	if n > 0 && !c.overflow {
		if c.buf.Len()+n > c.limit {
			c.overflow = true
			c.buf.Reset()
		} else {
			c.buf.Write(p[:n])
		}
	}
	return n, err
}

// sensitiveKeyParts 键名（小写、去掉 _ 与 -）含这些片段即视为凭据
var sensitiveKeyParts = []string{
	"password", "passwd", "secret", "token", "apikey", "accesskey", "privatekey",
	"credential", "authorization", "cookie", "webhook",
	"manifest", // 应用清单里可能有 Secret 的明文
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
	if runes := []rune(string(out)); len(runes) > maxAuditBodyLen {
		return string(runes[:maxAuditBodyLen]) + "..."
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

// AuditResource 审计资源常量
const (
	AuditResourceUser    = "user"
	AuditResourceCommand = "command"
	AuditResourceNotify  = "notify"
	AuditResourceCluster = "cluster"
)
