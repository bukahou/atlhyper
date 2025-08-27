// utils/gzip.go
//
// 包作用：
//
//	提供在 Agent ⇄ Master 之间复用的 gzip 压缩/解压工具，
//	覆盖两类常见场景：
//	  1) 内存中已有完整字节（如 Envelope 的 JSON）→ 压缩/解压成字节切片
//	  2) HTTP 服务器侧根据请求头 Content-Encoding 动态选择是否解压请求体
//
// 设计取舍：
//   - 面向「消息体规模：几十 KB ～ 数 MB」的常规使用，直接在内存完成压缩/解压，
//     实现简单、依赖少、足够快；若需要超大体积或流式传输，可在此基础上扩展 io.Pipe。
//   - MaybeGunzipReader 不修改业务逻辑，只是把“是否 gzip”这个判断聚合起来，
//     便于在接收端统一处理；注意它对 contentEncoding 的判断是严格等于 "gzip"。
//     如果上游可能传入诸如 "gzip, deflate" 的复合值，请在调用前自行解析并传入 "gzip"。
//   - 安全注意：务必在框架/Handler 层配合 MaxBytesReader（或等价限流）以防止大包/zip bomb。
//     解压后的读取也应再进行一次“解压后限流”（例如 io.LimitReader），避免放大攻击面。
package utils

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"strings"
)

// GzipBytes 将内存中的原始数据压缩为 gzip 格式的字节切片。
// 典型用途：Agent 侧在发送前对 Envelope 的 JSON 进行压缩，
// 并设置请求头 `Content-Encoding: gzip` 以降低网络传输体积。
//
// 压缩级别：使用 gzip.DefaultCompression（兼顾速度与压缩比）。
//   - 若更关注速度：可改为 gzip.BestSpeed
//   - 若更关注体积：可改为 gzip.BestCompression（CPU 成本更高）
//
// 性能与内存：
//   - 该函数会在内存中生成一份压缩后的新切片，适用于数 MB 级别以内的负载。
//   - 对于更大或 Streaming 场景，建议基于 gzip.Writer + io.Pipe 实现边写边压。
func GzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	// 写入原始数据
	if _, err := zw.Write(b); err != nil {
		_ = zw.Close()
		return nil, err
	}
	// 关闭以刷新尾部（CRC 等）
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MaybeGunzipReaderAuto：更通用的解压入口
// - 若 Content-Encoding 包含 gzip（大小写无关；容忍 "gzip, deflate"），则解压；
// - 否则嗅探前两个字节是否为 gzip 魔数 0x1f,0x8b，是则解压；
// - 上述都不是时，原样透传。
// 返回的 ReadCloser 需要调用方 Close()；Close 会级联关闭底层资源。
func MaybeGunzipReaderAuto(body io.ReadCloser, contentEncoding string) (io.ReadCloser, error) {
	// 1) 先看 Content-Encoding 是否包含 gzip（容忍 "gzip, deflate"）
	if hasGzipEncoding(contentEncoding) {
		zr, err := gzip.NewReader(body)
		if err != nil {
			_ = body.Close()
			return nil, err
		}
		return &compoundRC{r: zr, closers: []io.Closer{zr, body}}, nil
	}

	// 2) 头不可靠时，嗅探魔数（不消耗流，使用 Peek）
	br := bufio.NewReader(body)
	if sig, _ := br.Peek(2); len(sig) == 2 && sig[0] == 0x1f && sig[1] == 0x8b {
		zr, err := gzip.NewReader(br)
		if err != nil {
			_ = body.Close()
			return nil, err
		}
		return &compoundRC{r: zr, closers: []io.Closer{zr, body}}, nil
	}

	// 3) 既无头也非魔数：透传
	return &compoundRC{r: br, closers: []io.Closer{body}}, nil
}

// hasGzipEncoding 判断 Content-Encoding 是否包含 gzip（大小写无关、逗号分隔）
func hasGzipEncoding(v string) bool {
	if v == "" {
		return false
	}
	for _, t := range strings.Split(v, ",") {
		if strings.EqualFold(strings.TrimSpace(t), "gzip") {
			return true
		}
	}
	return false
}


// 复合 ReadCloser：
// - Read 从内部 reader 读取（可能是 gzip.Reader 或 bufio.Reader）
// - Close 依次关闭所有底层 io.Closer（先解压器，再原始 body）
type compoundRC struct {
	r       io.Reader
	closers []io.Closer
}

func (c *compoundRC) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *compoundRC) Close() error {
	var firstErr error
	for _, cl := range c.closers {
		if err := cl.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
