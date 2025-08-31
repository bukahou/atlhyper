package collect

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"AtlHyper/model/metrics"
)

var lastNetStats = make(map[string][2]uint64) // interface -> [rxBytes, txBytes]
var lastNetTime time.Time

// CollectNetwork 采集网络速率（KB/s + 可读格式）
func CollectNetwork() ([]metrics.NetworkStat, error) {
	devFile := filepath.Join(procRoot, "net/dev")
	file, err := os.Open(devFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	now := time.Now()
	var result []metrics.NetworkStat
	scanner := bufio.NewScanner(file)

	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()

		if i < 2 || !strings.Contains(line, ":") {
			continue
		}

		parts := strings.Split(line, ":")
		iface := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}

		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)

		// 若为第一次采集，仅记录
		if lastNetTime.IsZero() {
			lastNetStats[iface] = [2]uint64{rxBytes, txBytes}
			continue
		}

		last := lastNetStats[iface]
		deltaTime := now.Sub(lastNetTime).Seconds()
		rxKBps := float64(rxBytes-last[0]) / deltaTime / 1024
		txKBps := float64(txBytes-last[1]) / deltaTime / 1024

		// ✅ 本地格式化为可读字符串
		formatSpeed := func(kbps float64) string {
			if kbps >= 1024 {
				return fmt.Sprintf("%.2f MB/s", kbps/1024)
			}
			return fmt.Sprintf("%.2f KB/s", kbps)
		}

		result = append(result, metrics.NetworkStat{
			Interface: iface,
			RxKBps:    rxKBps,
			TxKBps:    txKBps,
			RxSpeed:   formatSpeed(rxKBps),
			TxSpeed:   formatSpeed(txKBps),
		})

		// 更新当前值
		lastNetStats[iface] = [2]uint64{rxBytes, txBytes}
	}

	lastNetTime = now
	return result, nil
}
