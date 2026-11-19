package database

import "time"

// Stamp 时间写进库、或拿来跟库里的时间比较时的统一格式：UTC 的 RFC 3339（秒精度，Z 结尾）。
// 库里的时间列都按这一种格式存（见 sqlite/schema/00002），字符串比较与排序才对。
func Stamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
