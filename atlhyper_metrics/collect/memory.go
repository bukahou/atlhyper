package collect

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"AtlHyper/model/metrics"
)

func CollectMemory() (metrics.MemoryStat, error) {
	var stat metrics.MemoryStat

	meminfoPath := filepath.Join(procRoot, "meminfo")
	file, err := os.Open(meminfoPath)
	if err != nil {
		return stat, err
	}
	defer file.Close()

	var memTotal, memAvailable uint64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := fields[0]
		value, _ := strconv.ParseUint(fields[1], 10, 64) // 单位：kB

		switch key {
		case "MemTotal:":
			memTotal = value
		case "MemAvailable:":
			memAvailable = value
		}
	}

	if err := scanner.Err(); err != nil {
		return stat, err
	}

	// ✅ 字节单位
	stat.Total = memTotal * 1024
	stat.Available = memAvailable * 1024
	stat.Used = stat.Total - stat.Available
	stat.Usage = float64(stat.Used) / float64(stat.Total)

	// ✅ 可读单位（自动保留 2 位小数）
	stat.TotalReadable = byteToHuman(stat.Total)
	stat.AvailableReadable = byteToHuman(stat.Available)
	stat.UsedReadable = byteToHuman(stat.Used)
	stat.UsagePercent = fmt.Sprintf("%.2f%%", stat.Usage*100)

	return stat, nil
}

// byteToHuman 将字节数转换为可读格式（GB 或 MB）
func byteToHuman(b uint64) string {
	gb := float64(b) / (1024 * 1024 * 1024)
	if gb >= 1 {
		return fmt.Sprintf("%.2f GB", gb)
	}
	mb := float64(b) / (1024 * 1024)
	return fmt.Sprintf("%.2f MB", mb)
}
