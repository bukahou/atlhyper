// Package httpserve 启动 HTTP 服务：先确认端口监听成功，再在后台提供服务。
//
// 监听失败要让调用方知道（进程启动失败），⛔ 不能只在后台协程里打一行日志、对外报成功。
package httpserve

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

// Start 监听 srv.Addr，成功后在后台 Serve；监听失败返回错误。
// 服务运行中出错（ErrServerClosed 以外）交给 onServeError。
func Start(srv *http.Server, onServeError func(error)) error {
	_, err := StartAddr(srv, onServeError)
	return err
}

// StartAddr 同 Start，并返回实际监听的地址（Addr 写 :0 时由系统分配端口）
func StartAddr(srv *http.Server, onServeError func(error)) (string, error) {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return "", fmt.Errorf("监听 %s: %w", srv.Addr, err)
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			onServeError(err)
		}
	}()
	return ln.Addr().String(), nil
}
