package httpserve

import (
	"net"
	"net/http"
	"testing"
)

// 端口被占用：启动返回错误（不能只在后台打一行日志、对外报成功）
func TestStart_ListenFailureIsAnError(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()

	srv := &http.Server{Addr: busy.Addr().String(), Handler: http.NotFoundHandler()}
	if err := Start(srv, func(error) {}); err == nil {
		t.Fatal("端口被占用时 Start 应返回错误")
	}
}

func TestStart_Serves(t *testing.T) {
	srv := &http.Server{Addr: "127.0.0.1:0", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})}
	addr, err := StartAddr(srv, func(error) {})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("status = %d", resp.StatusCode)
	}
}
