package gemini_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/option"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/llm/gemini"
)

// 测试用占位密钥（⛔ 不是真实密钥）
const redactTestKey = "test-key-placeholder-redact"

// REST 传输把 API Key 放在请求网址的 key 参数里：连不上时的错误带完整网址。
// 往上交出的错误（也是写进日志的那份）⛔ 不能带 API Key，分类照常是「连不上」
func TestChatStream_ErrorDoesNotLeakAPIKey(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "http://" + listener.Addr().String()
	listener.Close() // 关掉监听：之后连这个地址会被拒绝

	client, err := gemini.NewGeminiClient(redactTestKey, "gemini-redact", option.WithEndpoint(endpoint))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.ChatStream(ctx, &llm.Request{Messages: []llm.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		if strings.Contains(err.Error(), redactTestKey) {
			t.Fatalf("错误里带了 API Key: %v", err)
		}
		return
	}
	var streamErr error
	for chunk := range stream {
		if chunk.Type == llm.ChunkError {
			streamErr = chunk.Error
		}
	}
	if streamErr == nil {
		t.Fatal("连不上的地址应以错误收尾")
	}
	if strings.Contains(streamErr.Error(), redactTestKey) {
		t.Fatalf("错误里带了 API Key: %v", streamErr)
	}
	if kind := llm.Classify(streamErr); kind != llm.ErrKindUnreachable {
		t.Errorf("Classify = %q, want %q（抹掉密钥不能丢错误链）", kind, llm.ErrKindUnreachable)
	}
}
