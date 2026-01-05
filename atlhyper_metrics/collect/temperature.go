package collect

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"AtlHyper/atlhyper_metrics/config"
	"AtlHyper/model/collect"
)

// CollectTemperature 采集 CPU、GPU、NVMe 的温度（单位：℃）
func CollectTemperature() (collect.TemperatureStat, error) {
	var stat collect.TemperatureStat

	// 从配置获取 /sys 路径
	sysRoot := config.C.Collect.SysRoot
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

		// 匹配不同的设备类型（保持现有逻辑不变）
		switch name {
		case "k10temp", "coretemp":
			// CPU 温度
			if stat.CPUDegrees == 0 {
				if tempC, err := readFirstAvailableTemp(hwmonPath); err == nil {
					stat.CPUDegrees = tempC
				}
			}
		case "amdgpu":
			// GPU 温度
			if stat.GPUDegrees == 0 {
				if tempC, err := readFirstAvailableTemp(hwmonPath); err == nil {
					stat.GPUDegrees = tempC
				}
			}
		case "nvme":
			// NVMe 温度
			if stat.NVMEDegrees == 0 {
				if tempC, err := readFirstAvailableTemp(hwmonPath); err == nil {
					stat.NVMEDegrees = tempC
				}
			}
		// （可选）有些平台把 CPU 也暴露成这些名字，顺手兼容一下；不会影响 Ubuntu
		case "cpu_thermal", "cpu-thermal", "soc_thermal", "soc-thermal":
			if stat.CPUDegrees == 0 {
				if tempC, err := readFirstAvailableTemp(hwmonPath); err == nil {
					stat.CPUDegrees = tempC
				}
			}
		}
	}

	// ✅ 新增：只有当 CPU 还没取到时，才回退到 thermal_zone（树莓派）
	if stat.CPUDegrees == 0 {
		if cpu, err := readCpuFromThermalZones(sysRoot); err == nil && cpu > 0 {
			stat.CPUDegrees = cpu
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

// ✅ 新增函数：在 /sys/class/thermal/thermal_zone*/ 中查找 CPU 温度
func readCpuFromThermalZones(sysRoot string) (float64, error) {
	base := filepath.Join(sysRoot, "class/thermal")
	entries, err := os.ReadDir(base)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "thermal_zone") {
			continue
		}
		zonePath := filepath.Join(base, e.Name())
		typBytes, err := os.ReadFile(filepath.Join(zonePath, "type"))
		if err != nil {
			continue
		}
		typ := strings.TrimSpace(string(typBytes))
		// 树莓派常见类型
		if typ == "cpu-thermal" || typ == "cpu_thermal" || typ == "soc-thermal" || typ == "soc_thermal" {
			tb, err := os.ReadFile(filepath.Join(zonePath, "temp"))
			if err != nil {
				continue
			}
			if milli, err := strconv.Atoi(strings.TrimSpace(string(tb))); err == nil && milli > 0 {
				return float64(milli) / 1000.0, nil
			}
		}
	}
	return 0, os.ErrNotExist
}
