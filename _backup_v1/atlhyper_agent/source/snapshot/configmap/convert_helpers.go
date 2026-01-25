package configmap

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// fmtAge —— 与其它资源统一的“简洁时长”
func fmtAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	day := d / (24 * time.Hour)
	d -= day * 24 * time.Hour
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	switch {
	case day > 0:
		return fmt.Sprintf("%dd%dh", day, h)
	case h > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// pickAnnotations —— 只挑选少数常见注解，避免把超大注解（如 last-applied）带上
func pickAnnotations(ann map[string]string) map[string]string {
	if len(ann) == 0 {
		return nil
	}
	// 白名单（可按需增补）
	keys := []string{
		"app.kubernetes.io/name",
		"app.kubernetes.io/instance",
		"app.kubernetes.io/managed-by",
		"helm.sh/chart",
	}
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := ann[k]; ok && v != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// utf8ByteLen —— 返回 UTF-8 字符串占用的字节数
func utf8ByteLen(s string) int { return len([]byte(s)) }

// previewUTF8 —— 在不破坏 UTF-8 的前提下，按字节上限生成预览
func previewUTF8(s string, limit int) (string, bool) {
	if limit <= 0 {
		return "", s != ""
	}
	b := []byte(s)
	if len(b) <= limit {
		return s, false
	}
	// 不能简单截断字节，需回退到合法的 rune 边界
	cut := limit
	for cut > 0 && !utf8.FullRune(b[:cut]) {
		cut--
	}
	if cut <= 0 {
		// 极端情况，退回逐 rune 累计
		var acc int
		var sb strings.Builder
		for _, r := range s {
			sz := utf8.RuneLen(r)
			if acc+sz > limit {
				break
			}
			sb.WriteRune(r)
			acc += sz
		}
		return sb.String(), true
	}
	return string(b[:cut]), true
}
