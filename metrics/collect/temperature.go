package collect

import (
	"NeuroController/model/metrics"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CollectTemperature 采集 CPU、GPU、NVMe 的温度（单位：℃）
func CollectTemperature() (metrics.TemperatureStat, error) {
	var stat metrics.TemperatureStat

	// ✅ 支持通过环境变量覆盖宿主机 /sys 路径（默认使用容器内 /sys）
	sysRoot := os.Getenv("SYS_ROOT")
	if sysRoot == "" {
		sysRoot = "/sys"
	}
	baseDir := filepath.Join(sysRoot, "class/hwmon")

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return stat, err
	}

	for _, entry := range entries {
		hwmonPath := filepath.Join(baseDir, entry.Name())

		// 获取设备名称
		nameData, err := os.ReadFile(filepath.Join(hwmonPath, "name"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(nameData))

		// 匹配不同的设备类型
		switch name {
		case "k10temp", "coretemp":
			// CPU 温度
			tempC, err := readFirstAvailableTemp(hwmonPath)
			if err == nil {
				stat.CPUDegrees = tempC
			}
		case "amdgpu":
			// GPU 温度
			tempC, err := readFirstAvailableTemp(hwmonPath)
			if err == nil {
				stat.GPUDegrees = tempC
			}
		case "nvme":
			// NVMe 温度
			tempC, err := readFirstAvailableTemp(hwmonPath)
			if err == nil {
				stat.NVMEDegrees = tempC
			}
		}
	}

	return stat, nil
}

// 读取第一个 temp*_input 的温度值（单位转换为 ℃）
func readFirstAvailableTemp(hwmonPath string) (float64, error) {
	files, err := os.ReadDir(hwmonPath)
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "temp") && strings.HasSuffix(file.Name(), "_input") {
			data, err := os.ReadFile(filepath.Join(hwmonPath, file.Name()))
			if err != nil {
				continue
			}
			if milliDeg, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				return float64(milliDeg) / 1000.0, nil
			}
		}
	}
	return 0, os.ErrNotExist
}
