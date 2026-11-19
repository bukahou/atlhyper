package aiops

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	aiPkg "AtlHyper/atlhyper_master_v2/ai"
)

// 工具执行期间没有输出时也定时发心跳注释（防止中间的代理按空闲断开）；分段照常逐条发出
func TestStreamSSE_HeartbeatsWhileIdleAndRelaysChunks(t *testing.T) {
	ch := make(chan *aiPkg.ChatChunk)
	go func() {
		time.Sleep(60 * time.Millisecond) // 模拟一段没有输出的工具执行
		ch <- &aiPkg.ChatChunk{Type: "text", Content: "好"}
		ch <- &aiPkg.ChatChunk{Type: "done"}
		close(ch)
	}()
	rec := httptest.NewRecorder()
	streamSSE(rec, ch, 20*time.Millisecond)

	body := rec.Body.String()
	if !strings.Contains(body, ": ping") {
		t.Errorf("空闲期间应有心跳注释，实际:\n%s", body)
	}
	if !strings.Contains(body, `"content":"好"`) || !strings.Contains(body, `"type":"done"`) {
		t.Errorf("分段应原样发出，实际:\n%s", body)
	}
}

// 写失败（客户端已断开）也把通道读到关闭：否则产出方阻塞在发送上，协程泄漏
func TestStreamSSE_DrainsChannelAfterClientGone(t *testing.T) {
	ch := make(chan *aiPkg.ChatChunk)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			ch <- &aiPkg.ChatChunk{Type: "text", Content: "x"}
		}
		close(ch)
		close(done)
	}()
	streamSSE(failingWriter{httptest.NewRecorder()}, ch, time.Hour)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("客户端断开后没有把通道读完，产出方被卡住")
	}
}

type failingWriter struct{ *httptest.ResponseRecorder }

func (failingWriter) Write([]byte) (int, error) { return 0, errClientGone }

var errClientGone = errors.New("client gone")
