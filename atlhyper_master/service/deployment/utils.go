// ui_interfaces/deployment/utils.go
package deployment

import (
	modelpod "AtlHyper/model/k8s"
)

// mapLen —— 安全获取 map 长度
func mapLen(m map[string]string) int {
	if m == nil {
		return 0
	}
	return len(m)
}

// joinImagesShort —— 取第一个容器镜像；多容器时返回 "img1 (+N)"
func joinImagesShort(cs []modelpod.Container) string {
	if len(cs) == 0 {
		return ""
	}
	first := cs[0].Image
	if len(cs) == 1 {
		return first
	}
	return first + " (+" + itoa(len(cs)-1) + ")"
}

func itoa(i int) string {
	// 避免引入 strconv，简单实现
	if i == 0 {
		return "0"
	}
	d := [20]byte{}
	pos := len(d)
	n := i
	for n > 0 {
		pos--
		d[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(d[pos:])
}
