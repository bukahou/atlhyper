package llmtest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

const (
	streamTimeout = 10 * time.Second // 一次流从发请求到通道关闭的上限
	exitTimeout   = 5 * time.Second  // 取消后适配层 goroutine 退出的上限
	fillTimeout   = 3 * time.Second  // 等适配层写满通道缓冲的上限
	settleTimeout = time.Second      // 等上一个用例的 goroutine 退干净的上限
)

// RunContract 对一家适配层跑全部契约用例。
// 用例依次执行、不并行：取消用例要数适配层自己的 goroutine，别的用例同时在跑会数混。
func RunContract(t *testing.T, a Adapter) {
	t.Run("文本_分段按序到达并以完成收尾", func(t *testing.T) { checkTextReply(t, a) })
	t.Run("工具调用_参数分多段到达", func(t *testing.T) { checkFragmentedToolCall(t, a) })
	t.Run("工具调用_参数一次到达", func(t *testing.T) { checkWholeToolCall(t, a) })
	t.Run("HTTP401_认证类", func(t *testing.T) { checkHTTPError(t, a, http.StatusUnauthorized, llm.ErrKindAuth) })
	t.Run("HTTP429_限流类", func(t *testing.T) { checkHTTPError(t, a, http.StatusTooManyRequests, llm.ErrKindRateLimited) })
	t.Run("响应体在结束标志前结束_未完成", func(t *testing.T) { checkTruncated(t, a) })
	t.Run("连接在结束标志前断开_报错", func(t *testing.T) { checkConnectionCut(t, a) })
	t.Run("中途取消_按取消处理", func(t *testing.T) { checkCancelMidStream(t, a) })
	t.Run("发请求前已取消_按取消处理", func(t *testing.T) { checkCanceledBeforeRequest(t, a) })
}

// 1. 纯文本：分段按序到达，以完成收尾并带用量
func checkTextReply(t *testing.T, a Adapter) {
	reply := Reply{
		Texts:    []string{"default 里", "有 3 个 Pod，", "都在运行。"},
		Finished: true,
		Usage:    llm.Usage{InputTokens: 120, OutputTokens: 18},
	}
	checkCompleted(t, a, reply)
}

// 2a. 工具参数分多段到达（切在键名中间），拼成一个完整的工具调用
func checkFragmentedToolCall(t *testing.T, a Adapter) {
	if a.ToolArgsInOnePiece != "" {
		t.Skip("不适用：" + a.ToolArgsInOnePiece)
	}
	reply := Reply{
		Texts: []string{"先查一下 Pod。"},
		ToolCall: &ToolCallReply{
			ID:           "call_contract_01",
			Name:         "get_pods",
			ArgFragments: []string{`{"namespace":`, `"default","li`, `mit":5}`},
		},
		Finished: true,
		Usage:    llm.Usage{InputTokens: 230, OutputTokens: 31},
	}
	checkCompleted(t, a, reply)
}

// 2b. 工具参数一次到达（没有文字）
func checkWholeToolCall(t *testing.T, a Adapter) {
	reply := Reply{
		ToolCall: &ToolCallReply{
			ID:           "call_contract_02",
			Name:         "get_events",
			ArgFragments: []string{`{"namespace":"kube-system","limit":20}`},
		},
		Finished: true,
		Usage:    llm.Usage{InputTokens: 260, OutputTokens: 12},
	}
	checkCompleted(t, a, reply)
}

func checkCompleted(t *testing.T, a Adapter, reply Reply) {
	up := startUpstream(t, a.Wire.Stream(reply), a.TLS)
	chunks, err := streamAll(t, context.Background(), a, up)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	report(t, completedProblems(chunks, reply, a.ToolCallIDOnWire))
	checkRequestPaths(t, up, a.Path)
}

// 3. 上游 HTTP 错误按状态码分类，⛔ 不能以完成收尾
func checkHTTPError(t *testing.T, a Adapter, status int, want llm.ErrorKind) {
	up := startUpstream(t, a.Wire.Error(status), a.TLS)
	chunks, err := streamAll(t, context.Background(), a, up)
	report(t, failedProblems(err, chunks, want))
	if texts := textsOf(chunks); len(texts) > 0 {
		t.Errorf("HTTP %d 时不该有文字，实际 %q", status, texts)
	}
	checkRequestPaths(t, up, a.Path)
}

// 4a. 响应体正常结束，但内容里没有结束标志：流未完成，⛔ 不能当成功
func checkTruncated(t *testing.T, a Adapter) {
	up := startUpstream(t, a.Wire.Stream(Reply{Texts: []string{"Pod 列表如下：", "web-1 运行中，"}}), a.TLS)
	chunks, err := streamAll(t, context.Background(), a, up)
	report(t, failedProblems(err, chunks, llm.ErrKindIncomplete))
	checkRequestPaths(t, up, a.Path)
}

// 4b. 结束标志之前连接被掐断：报错，⛔ 不能当成功。
// 连接中断更贴切，未完成也如实（SDK 把半截响应体当作读完时，适配层按缺结束标志报未完成）。
func checkConnectionCut(t *testing.T, a Adapter) {
	resp := a.Wire.Stream(Reply{Texts: []string{"Pod 列表如下：", "web-1 运行中，"}})
	resp.Ending = CutConnection
	up := startUpstream(t, resp, a.TLS)
	chunks, err := streamAll(t, context.Background(), a, up)
	report(t, failedProblems(err, chunks, llm.ErrKindConnectionLost, llm.ErrKindIncomplete))
	checkRequestPaths(t, up, a.Path)
}

// 5a. 回答到一半调用方取消：首段即时送到；取消后没人再读，适配层也要退出；⛔ 不以完成收尾
func checkCancelMidStream(t *testing.T, a Adapter) {
	texts := make([]string, 200) // 多于任何一家的通道缓冲，让适配层在取消时正卡在发送上
	for i := range texts {
		texts[i] = fmt.Sprintf("第 %d 段；", i+1)
	}
	resp := a.Wire.Stream(Reply{Texts: texts})
	resp.Ending = HoldOpen
	up := startUpstream(t, resp, a.TLS)
	client := a.NewClient(t, up.endpoint)
	t.Cleanup(func() { _ = client.Close() })
	pkg := packageOf(client)
	baseline := settledGoroutinesIn(pkg, settleTimeout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.ChatStream(ctx, contractRequest(t))
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	// 上游还没收尾，首段就要送到：流式转发，不攒到最后
	first := receiveOne(t, stream)
	if first == nil || first.Type != llm.ChunkText || first.Content != texts[0] {
		t.Fatalf("首段应为文字 %q，实际 %+v", texts[0], first)
	}

	// 调用方暂不读，等适配层把通道缓冲写满，再取消
	waitChannelFull(stream, fillTimeout)
	cancel()

	// 取消后调用方不再读（ReadStream 直接返回）：适配层的 goroutine 不能卡在发送上
	if stuck := waitGoroutinesIn(pkg, baseline, exitTimeout); stuck != nil {
		t.Errorf("取消后没人再读，%s 起的 goroutine %v 内没有退出（连同上游连接一起泄漏），卡在：\n%s",
			pkg, exitTimeout, strings.Join(stuck, "\n"))
	}
	report(t, canceledProblems(nil, collectChunks(t, stream)))
	checkRequestPaths(t, up, a.Path)
}

// 5b. 发请求前 ctx 已取消
func checkCanceledBeforeRequest(t *testing.T, a Adapter) {
	up := startUpstream(t, a.Wire.Stream(Reply{
		Texts:    []string{"不该送到"},
		Finished: true,
		Usage:    llm.Usage{InputTokens: 1, OutputTokens: 1},
	}), a.TLS)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chunks, err := streamAll(t, ctx, a, up)
	report(t, canceledProblems(err, chunks))
}

// contractRequest 每个用例发的请求。假上游不看它；带上工具定义，让各家的请求转换也走一遍。
func contractRequest(t *testing.T) *llm.Request {
	t.Helper()
	schema, err := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"namespace": map[string]any{"type": "string"},
			"limit":     map[string]any{"type": "integer"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &llm.Request{
		SystemPrompt: "你是集群运维助手。",
		Messages:     []llm.Message{{Role: "user", Content: "default 命名空间里的 Pod 怎么样？"}},
		Tools:        []llm.ToolDefinition{{Name: "get_pods", Description: "列出 Pod", Parameters: schema}},
	}
}

// streamAll 建客户端、发一次请求并读到通道关闭。ChatStream 报错时返回该错误（没有分段）。
func streamAll(t *testing.T, ctx context.Context, a Adapter, up *upstream) ([]*llm.Chunk, error) {
	t.Helper()
	client := a.NewClient(t, up.endpoint)
	t.Cleanup(func() { _ = client.Close() })
	stream, err := client.ChatStream(ctx, contractRequest(t))
	if err != nil {
		return nil, err
	}
	return collectChunks(t, stream), nil
}

// collectChunks 读到通道关闭为止；超时说明适配层没关通道
func collectChunks(t *testing.T, stream <-chan *llm.Chunk) []*llm.Chunk {
	t.Helper()
	var chunks []*llm.Chunk
	timeout := time.After(streamTimeout)
	for {
		select {
		case c, ok := <-stream:
			if !ok {
				return chunks
			}
			chunks = append(chunks, c)
		case <-timeout:
			t.Fatalf("%v 内通道没有关闭（已收到 %d 段）", streamTimeout, len(chunks))
		}
	}
}

func receiveOne(t *testing.T, stream <-chan *llm.Chunk) *llm.Chunk {
	t.Helper()
	select {
	case c := <-stream:
		return c
	case <-time.After(streamTimeout):
		t.Fatalf("%v 内没收到首段：上游还没收尾时就该逐段转发", streamTimeout)
		return nil
	}
}

// waitChannelFull 等通道缓冲写满（缓冲比上游给的分段还大时等到超时为止）
func waitChannelFull(stream <-chan *llm.Chunk, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for len(stream) < cap(stream) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
}

func checkRequestPaths(t *testing.T, up *upstream, want string) {
	t.Helper()
	paths := up.requestPaths()
	if len(paths) == 0 {
		t.Error("假上游没收到请求")
	}
	for _, p := range paths {
		if p != want {
			t.Errorf("请求路径为 %q，应为 %q", p, want)
		}
	}
}

func report(t *testing.T, problems []string) {
	t.Helper()
	for _, p := range problems {
		t.Error(p)
	}
}
