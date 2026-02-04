package collector

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/model_v2"
)

// temperatureCollector 温度采集器实现
type temperatureCollector struct {
	cfg *config.Config

	mu      sync.RWMutex
	metrics model_v2.TemperatureMetrics
}

// NewTemperatureCollector 创建温度采集器
func NewTemperatureCollector(cfg *config.Config) TemperatureCollector {
	return &temperatureCollector{
		cfg: cfg,
	}
}

// Collect 采集温度指标
func (c *temperatureCollector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = model_v2.TemperatureMetrics{}

	// 从 hwmon 读取所有传感器
	c.readHwmon()

	// 如果还没有 CPU 温度，回退到 thermal_zone（树莓派等）
	if c.metrics.CPUTemp == 0 {
		c.readThermalZone()
	}

	return nil
}

// Get 获取温度指标
func (c *temperatureCollector) Get() model_v2.TemperatureMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// readHwmon 从 /sys/class/hwmon 读取温度
func (c *temperatureCollector) readHwmon() {
	baseDir := filepath.Join(c.cfg.Paths.SysRoot, "class/hwmon")

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return
	}

	var sensors []model_v2.SensorReading

	for _, entry := range entries {
		hwmonPath := filepath.Join(baseDir, entry.Name())

		// 读取传感器名称
		nameData, err := os.ReadFile(filepath.Join(hwmonPath, "name"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(nameData))

		// 读取所有温度传感器
		files, err := os.ReadDir(hwmonPath)
		if err != nil {
			continue
		}

		for _, file := range files {
			if !strings.HasPrefix(file.Name(), "temp") || !strings.HasSuffix(file.Name(), "_input") {
				continue
			}

			// 提取编号 (temp1_input -> 1)
			num := strings.TrimSuffix(strings.TrimPrefix(file.Name(), "temp"), "_input")
			prefix := "temp" + num

			// 读取当前温度 (毫摄氏度 -> 摄氏度)
			tempData, err := os.ReadFile(filepath.Join(hwmonPath, file.Name()))
			if err != nil {
				continue
			}
			milliC, err := strconv.Atoi(strings.TrimSpace(string(tempData)))
			if err != nil {
				continue
			}
			current := float64(milliC) / 1000.0

			// 读取标签
			label := name + " " + num
			if labelData, err := os.ReadFile(filepath.Join(hwmonPath, prefix+"_label")); err == nil {
				label = strings.TrimSpace(string(labelData))
			}

			// 读取最高阈值
			var max float64
			if maxData, err := os.ReadFile(filepath.Join(hwmonPath, prefix+"_max")); err == nil {
				if maxMilliC, err := strconv.Atoi(strings.TrimSpace(string(maxData))); err == nil {
					max = float64(maxMilliC) / 1000.0
				}
			}

			// 读取临界阈值
			var critical float64
			if critData, err := os.ReadFile(filepath.Join(hwmonPath, prefix+"_crit")); err == nil {
				if critMilliC, err := strconv.Atoi(strings.TrimSpace(string(critData))); err == nil {
					critical = float64(critMilliC) / 1000.0
				}
			}

			sensor := model_v2.SensorReading{
				Name:     name,
				Label:    label,
				Current:  current,
				Max:      max,
				Critical: critical,
			}
			sensors = append(sensors, sensor)

			// 识别 CPU 温度
			if c.isCPUSensor(name) && current > c.metrics.CPUTemp {
				c.metrics.CPUTemp = current
				if critical > 0 {
					c.metrics.CPUTempMax = critical
				} else if max > 0 {
					c.metrics.CPUTempMax = max
				} else {
					// 默认最高温度阈值
					c.metrics.CPUTempMax = 100
				}
			}
		}
	}

	c.metrics.Sensors = sensors
}

// readThermalZone 从 /sys/class/thermal/thermal_zone* 读取温度
// 用于树莓派等没有 hwmon CPU 传感器的设备
func (c *temperatureCollector) readThermalZone() {
	baseDir := filepath.Join(c.cfg.Paths.SysRoot, "class/thermal")

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "thermal_zone") {
			continue
		}

		zonePath := filepath.Join(baseDir, entry.Name())

		// 读取类型
		typeData, err := os.ReadFile(filepath.Join(zonePath, "type"))
		if err != nil {
			continue
		}
		zoneType := strings.TrimSpace(string(typeData))

		// 读取温度 (毫摄氏度)
		tempData, err := os.ReadFile(filepath.Join(zonePath, "temp"))
		if err != nil {
			continue
		}
		milliC, err := strconv.Atoi(strings.TrimSpace(string(tempData)))
		if err != nil || milliC <= 0 {
			continue
		}
		current := float64(milliC) / 1000.0

		// 添加到传感器列表
		sensor := model_v2.SensorReading{
			Name:    "thermal_zone",
			Label:   zoneType,
			Current: current,
		}
		c.metrics.Sensors = append(c.metrics.Sensors, sensor)

		// 识别 CPU 温度（树莓派常见类型）
		if c.isCPUThermalZone(zoneType) && current > c.metrics.CPUTemp {
			c.metrics.CPUTemp = current
			c.metrics.CPUTempMax = 85 // 树莓派默认阈值
		}
	}

	// 如果还没有 CPU 温度，取第一个传感器
	if c.metrics.CPUTemp == 0 && len(c.metrics.Sensors) > 0 {
		c.metrics.CPUTemp = c.metrics.Sensors[0].Current
		c.metrics.CPUTempMax = 85
	}
}

// isCPUSensor 判断是否为 CPU 温度传感器
func (c *temperatureCollector) isCPUSensor(name string) bool {
	// AMD / Intel / 常见主板传感器
	cpuSensors := []string{
		"k10temp",   // AMD
		"coretemp",  // Intel
		"zenpower",  // AMD Zen
		"cpu_thermal", // 一些平台
		"cpu-thermal",
		"soc_thermal",
		"soc-thermal",
	}

	nameLower := strings.ToLower(name)
	for _, s := range cpuSensors {
		if nameLower == s {
			return true
		}
	}
	return false
}

// isCPUThermalZone 判断是否为 CPU thermal_zone
func (c *temperatureCollector) isCPUThermalZone(zoneType string) bool {
	// 树莓派 / ARM 常见类型
	cpuTypes := []string{
		"cpu-thermal",
		"cpu_thermal",
		"soc-thermal",
		"soc_thermal",
		"x86_pkg_temp",
		"acpitz",
	}

	typeLower := strings.ToLower(zoneType)
	for _, t := range cpuTypes {
		if typeLower == t {
			return true
		}
	}
	return false
}
