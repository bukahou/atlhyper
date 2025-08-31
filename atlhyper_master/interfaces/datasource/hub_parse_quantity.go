// interfaces/datasource/hub_parse_quantity.go
package datasource

import "fmt"

// 如需把 "32Gi" / "256Mi" / "8" 解析为字节或核，可在这里扩展。
// 先留占位，避免到处散落解析逻辑；后续可替换为 resource.ParseQuantity。

func parseCores(s string) float64 {
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}

func parseBytes(s string) uint64 {
	var n float64
	var unit string
	fmt.Sscanf(s, "%f%s", &n, &unit)
	switch unit {
	case "Gi":
		return uint64(n * 1024 * 1024 * 1024)
	case "Mi":
		return uint64(n * 1024 * 1024)
	case "":
		return uint64(n)
	default:
		return 0
	}
}
