package aiops

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	aiPkg "AtlHyper/atlhyper_master_v2/ai"
)

const (
	// sseHeartbeatInterval 没有分段输出时发一条注释的间隔：工具执行可能持续几十秒，
	// 中间的代理（Next、网关）会按空闲把连接断掉
	sseHeartbeatInterval = 15 * time.Second
	// sseWriteTimeout 每次写之前把写超时续到这么久之后：服务器的全局写超时管不了长时间的流
	sseWriteTimeout = 60 * time.Second
)

// streamSSE 把对话分段以 SSE 写给客户端，直到通道关闭。
// 写失败（客户端已断开）后照样把通道读完：产出方的发送不能被卡住。
func streamSSE(w http.ResponseWriter, ch <-chan *aiPkg.ChatChunk, heartbeat time.Duration) int {
	rc := http.NewResponseController(w)
	clientGone := false
	write := func(payload string) {
		if clientGone {
			return
		}
		_ = rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)) // 不支持时退回服务器的全局写超时
		if _, err := fmt.Fprint(w, payload); err != nil {
			clientGone = true
			return
		}
		if err := rc.Flush(); err != nil {
			clientGone = true
		}
	}

	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()
	chunks := 0
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				return chunks
			}
			chunks++
			data, err := json.Marshal(chunk)
			if err != nil {
				data, _ = json.Marshal(&aiPkg.ChatChunk{Type: "error", Content: "分段无法编码"})
			}
			if chunk.Type == "error" {
				aiLog.Warn("SSE 发送错误", "chunk", chunks, "type", chunk.Type)
			}
			write(fmt.Sprintf("event: message\ndata: %s\n\n", data))
			ticker.Reset(heartbeat)
		case <-ticker.C:
			write(": ping\n\n")
		}
	}
}
