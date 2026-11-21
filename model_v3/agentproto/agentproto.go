// Package agentproto 主控 ↔ 代理之间的协议版本。两端共用这一份常量：
// 代理每个请求都带版本头，主控只接受当前版本，不认识的明确拒绝（⛔ 不按零值继续处理）。
package agentproto

import "strconv"

const (
	// Header 协议版本头
	Header = "X-AtlHyper-Protocol"
	// Version 当前协议版本。快照 / 心跳 / 指令 / 结果的形状有不兼容的变化时加一
	Version = 1
)

// HeaderValue 请求里写的版本头取值
func HeaderValue() string { return strconv.Itoa(Version) }

// Accepts 主控是否接受这个版本头取值（只认当前版本）
func Accepts(value string) bool { return value == HeaderValue() }
