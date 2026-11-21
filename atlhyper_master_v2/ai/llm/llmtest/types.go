// Package llmtest 各家大模型适配层共用的契约测试：同一组用例对每家实现各跑一遍。
//
// 每家只需提供两样：自己的线上格式（Wire）和指向假上游建客户端的方法（Adapter.NewClient）。
// 假上游按用例预设的响应逐帧回放，⛔ 不看请求内容。
package llmtest

import (
	"net/http"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// ToolCallReply 上游回答里的一次工具调用
type ToolCallReply struct {
	ID           string
	Name         string
	ArgFragments []string // 依次拼起来是一个 JSON 对象；线上格式不分段的厂商把它们拼好一次给出
}

// Args 拼好的完整参数
func (c ToolCallReply) Args() string { return strings.Join(c.ArgFragments, "") }

// Reply 一次流式回答（与厂商无关）。停止原因由内容决定：有工具调用为「工具调用」，否则为「正常说完」。
type Reply struct {
	Texts    []string       // 文字分段，按到达顺序
	ToolCall *ToolCallReply // 文字之后的工具调用（可无）
	// Finished 写出收尾：块结束、停止原因、用量、结束标志。
	// false 时只写到最后一段内容为止，之后什么都不写（模拟上游没按约定收尾）。
	Finished bool
	Usage    llm.Usage // 收尾里带的用量
}

// Ending 写完全部帧之后假上游怎么结束这次响应
type Ending int

const (
	EndResponse   Ending = iota // 正常结束 HTTP 响应（响应体完整；内容里有没有结束标志由帧决定）
	CutConnection               // 直接断开 TCP 连接：分块传输没有结尾，客户端读到的响应体不完整
	HoldOpen                    // 不再写也不结束，直到客户端断开
)

// Response 假上游对每个请求回放的同一份 HTTP 响应
type Response struct {
	Status int // 0 按 200
	Header http.Header
	Frames [][]byte // 逐帧写出，每帧之后立即 Flush
	Ending Ending
}

// Wire 一家厂商的线上格式
type Wire interface {
	// Stream 把一次流式回答写成该厂商的 200 响应（Ending 由用例另设）
	Stream(reply Reply) Response
	// Error 该厂商对 status 的错误响应（错误体按该厂商的格式）
	Error(status int) Response
}

// Endpoint 假上游的地址；走 HTTPS 时 HTTPClient 信任假上游的证书
type Endpoint struct {
	BaseURL    string
	HTTPClient *http.Client
}

// Adapter 被测的一家适配层
type Adapter struct {
	// NewClient 建一个指向假上游的客户端（密钥用明显的占位值）
	NewClient func(t *testing.T, upstream Endpoint) llm.LLMClient
	Wire      Wire
	// TLS 假上游走 HTTPS（SDK 只肯经 HTTPS 发 Bearer 令牌的厂商）
	TLS bool
	// Path 该厂商流式接口的请求路径（转义后的形式），核对客户端确实按该厂商的接口在调
	Path string
	// ToolCallIDOnWire 线上格式带工具调用 ID：适配层必须原样透传；
	// false = 线上格式没有 ID，适配层自己生成，只要求非空
	ToolCallIDOnWire bool
	// ToolArgsInOnePiece 不为空 = 该厂商线上格式里工具参数总是一次给全（写明依据），
	// 「参数分多段到达」用例对它不适用，跳过并输出这条原因
	ToolArgsInOnePiece string
}
