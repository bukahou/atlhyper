package llmtest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// upstream 假上游：对每个请求回放同一份预设响应，只记下请求路径（⛔ 不看请求内容，不据此改变回应）
type upstream struct {
	endpoint Endpoint

	mu    sync.Mutex
	paths []string
}

// startUpstream 起一个按 resp 回放的假上游（useTLS 时走 HTTPS，仍是 HTTP/1.1），测试结束时关闭
func startUpstream(t *testing.T, resp Response, useTLS bool) *upstream {
	t.Helper()
	up := &upstream{}
	release := make(chan struct{}) // 测试结束时放开仍挂着的响应
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		up.mu.Lock()
		up.paths = append(up.paths, r.URL.EscapedPath())
		up.mu.Unlock()
		replay(w, r, resp, release)
	}))
	if useTLS {
		server.StartTLS()
	} else {
		server.Start()
	}
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) }) // 先于 server.Close 执行（Cleanup 后进先出）
	up.endpoint = Endpoint{BaseURL: server.URL, HTTPClient: server.Client()}
	return up
}

// replay 写出一次响应：每帧之后 Flush，让客户端逐帧收到；最后按 Ending 收尾
func replay(w http.ResponseWriter, r *http.Request, resp Response, release <-chan struct{}) {
	for key, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	controller := http.NewResponseController(w)
	for _, frame := range resp.Frames {
		if _, err := w.Write(frame); err != nil {
			return
		}
		if err := controller.Flush(); err != nil {
			return
		}
	}
	switch resp.Ending {
	case CutConnection:
		conn, _, err := controller.Hijack()
		if err == nil {
			_ = conn.Close()
		}
	case HoldOpen:
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}
}

// requestPaths 收到过的请求路径
func (u *upstream) requestPaths() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]string(nil), u.paths...)
}
