package push

import (
	"bytes"
	"context"
	"net/http"
)

// doPostGzipJSON 负责将 GZIP 压缩后的 JSON 数据通过 HTTP POST 发送到目标 URL。
// - ctx：用于控制请求生命周期（可超时/取消）
// - client：外部传入的 HTTP 客户端（已配置超时等）
// - url：接收端 URL（通常是 Agent 暴露的 Push 接口）
// - gzBody：已 GZIP 压缩的 JSON 数据
// - token：可选的 Bearer Token（用于认证）
//
// 返回：
// - nil：请求成功（HTTP 状态码 2xx）
// - error：请求构建失败、网络错误、非 2xx 状态码
func doPostGzipJSON(ctx context.Context, client *http.Client, url string, gzBody []byte, token string) error {
	// 构造带 Context 的 POST 请求，Body 使用压缩后的字节流
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(gzBody))
	if err != nil {
		// 创建请求对象失败（一般是 URL 无效等）
		return err
	}

	// 设置 HTTP 头：
	// Content-Type 声明原始数据是 JSON
	req.Header.Set("Content-Type", "application/json")
	// Content-Encoding 声明当前传输内容已 gzip 压缩
	req.Header.Set("Content-Encoding", "gzip")

	// 如果配置了 token，则加上 Bearer Token 认证头
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		// 网络层或连接错误（TCP 超时、DNS 失败等）
		return err
	}
	defer resp.Body.Close()

	// 业务规则：只要是 2xx 状态码就视为成功
	// resp.StatusCode/100 == 2 说明状态码是 200~299
	if resp.StatusCode/100 != 2 {
		// 返回自定义 httpError，包含状态码
		return &httpError{StatusCode: resp.StatusCode}
	}
	return nil
}

// httpError 用于包装非 2xx 状态码的错误
// 例如返回 500，则 httpError.Error() 会返回 "Internal Server Error"
type httpError struct{ StatusCode int }

// Error 实现 error 接口，返回 HTTP 状态码对应的文本描述
func (e *httpError) Error() string {
	return http.StatusText(e.StatusCode)
}
