// package collect

// import (
// 	"bufio"
// 	"fmt"
// 	"os"
// 	"path/filepath"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"AtlHyper/model/collect"
// )

// var lastNetStats = make(map[string][2]uint64) // interface -> [rxBytes, txBytes]
// var lastNetTime time.Time

// // CollectNetwork 采集网络速率（KB/s + 可读格式）
// func CollectNetwork() ([]collect.NetworkStat, error) {
// 	devFile := filepath.Join(ProcRoot(), "net/dev")
// 	file, err := os.Open(devFile)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	now := time.Now()
// 	var result []collect.NetworkStat
// 	scanner := bufio.NewScanner(file)

// 	for i := 0; scanner.Scan(); i++ {
// 		line := scanner.Text()

// 		if i < 2 || !strings.Contains(line, ":") {
// 			continue
// 		}

// 		parts := strings.Split(line, ":")
// 		iface := strings.TrimSpace(parts[0])
// 		fields := strings.Fields(parts[1])
// 		if len(fields) < 9 {
// 			continue
// 		}

// 		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
// 		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)

// 		// 若为第一次采集，仅记录
// 		if lastNetTime.IsZero() {
// 			lastNetStats[iface] = [2]uint64{rxBytes, txBytes}
// 			continue
// 		}

// 		last := lastNetStats[iface]
// 		deltaTime := now.Sub(lastNetTime).Seconds()
// 		rxKBps := float64(rxBytes-last[0]) / deltaTime / 1024
// 		txKBps := float64(txBytes-last[1]) / deltaTime / 1024

// 		// ✅ 本地格式化为可读字符串
// 		formatSpeed := func(kbps float64) string {
// 			if kbps >= 1024 {
// 				return fmt.Sprintf("%.2f MB/s", kbps/1024)
// 			}
// 			return fmt.Sprintf("%.2f KB/s", kbps)
// 		}

// 		result = append(result, collect.NetworkStat{
// 			Interface: iface,
// 			RxKBps:    rxKBps,
// 			TxKBps:    txKBps,
// 			RxSpeed:   formatSpeed(rxKBps),
// 			TxSpeed:   formatSpeed(txKBps),
// 		})

// 		// 更新当前值
// 		lastNetStats[iface] = [2]uint64{rxBytes, txBytes}
// 	}

// 	lastNetTime = now
// 	return result, nil
// }

package collect

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"AtlHyper/model/collect"
)

var lastNetStats = make(map[string][2]uint64) // interface -> [rxBytes, txBytes]
var lastNetTime time.Time

// detectMainInterface —— 通过 /proc/net/route 检测默认路由所用接口
func detectMainInterface() string {
	routeFile := filepath.Join(ProcRoot(), "net/route")
	f, err := os.Open(routeFile)
	if err != nil {
		return "" // 回退
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Iface") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == "00000000" { // 0.0.0.0 表示默认网关
			return fields[0]
		}
	}
	return ""
}

// CollectNetwork —— 采集主接口网络速率（统一上报为 eth0）
func CollectNetwork() ([]collect.NetworkStat, error) {
	devFile := filepath.Join(ProcRoot(), "net/dev")
	file, err := os.Open(devFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	mainIface := detectMainInterface()
	if mainIface == "" {
		mainIface = "eth0" // 默认回退
	}

	now := time.Now()
	scanner := bufio.NewScanner(file)
	var result []collect.NetworkStat

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

		// 只处理主接口（例如 eno1 / wlan0 / enp2s0 等）
		if iface != mainIface {
			continue
		}

		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)

		// 第一次采集仅记录
		if lastNetTime.IsZero() {
			lastNetStats[iface] = [2]uint64{rxBytes, txBytes}
			continue
		}

		last := lastNetStats[iface]
		deltaTime := now.Sub(lastNetTime).Seconds()
		if deltaTime <= 0 {
			continue
		}

		rxKBps := float64(rxBytes-last[0]) / deltaTime / 1024
		txKBps := float64(txBytes-last[1]) / deltaTime / 1024

		formatSpeed := func(kbps float64) string {
			if kbps >= 1024 {
				return fmt.Sprintf("%.2f MB/s", kbps/1024)
			}
			return fmt.Sprintf("%.2f KB/s", kbps)
		}

		// ✅ 上报时强制命名为 eth0
		result = append(result, collect.NetworkStat{
			Interface: "eth0",
			RxKBps:    rxKBps,
			TxKBps:    txKBps,
			RxSpeed:   formatSpeed(rxKBps),
			TxSpeed:   formatSpeed(txKBps),
		})

		lastNetStats[iface] = [2]uint64{rxBytes, txBytes}
	}

	lastNetTime = now
	return result, nil
}
