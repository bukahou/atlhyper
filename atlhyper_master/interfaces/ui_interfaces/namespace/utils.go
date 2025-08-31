package namespace

// 返回 map 的长度，nil map 返回 0
func mapLen(m map[string]string) int {
	if m == nil {
		return 0
	}

	return len(m)
}