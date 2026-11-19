// Package truncate 按字符截断文本：按字节切会把多字节字符（中日文）切成乱码。全项目只用这一份。
package truncate

// Runes s 超过 max 个字符时截到 max 个并接上 suffix；不超过原样返回
func Runes(s string, max int, suffix string) string {
	if max < 0 {
		max = 0
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i] + suffix
		}
		n++
	}
	return s
}
