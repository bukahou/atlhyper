package node

import (
	"strconv"
	"strings"
)

func atoiSafe(s string) int {
	i, _ := strconv.Atoi(strings.TrimSpace(s))
	return i
}

// "8" or "8.0" -> 8
func parseCPUToInt(v string) int {
	if v == "" {
		return 0
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return int(f + 1e-9)
	}
	return 0
}

// "3500m" -> 3.5 ; "2" -> 2
func parseCPUUsageToCores(v string) float64 {
	s := strings.TrimSpace(v)
	if s == "" {
		return 0
	}
	if strings.HasSuffix(s, "m") {
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "m"), 64)
		return f / 1000.0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// 支持 Ki/Mi/Gi/Ti 以及字节数
func parseMemToGiB(v string) float64 {
	s := strings.TrimSpace(v)
	if s == "" {
		return 0
	}
	switch {
	case strings.HasSuffix(s, "Ki"):
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "Ki"), 64)
		return f / 1024.0 / 1024.0
	case strings.HasSuffix(s, "Mi"):
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "Mi"), 64)
		return f / 1024.0
	case strings.HasSuffix(s, "Gi"):
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "Gi"), 64)
		return f
	case strings.HasSuffix(s, "Ti"):
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "Ti"), 64)
		return f * 1024.0
	default:
		f, _ := strconv.ParseFloat(s, 64) // bytes
		return f / (1024.0 * 1024.0 * 1024.0)
	}
}

func round1(x float64) float64 { return float64(int(x*10+0.5)) / 10.0 }
