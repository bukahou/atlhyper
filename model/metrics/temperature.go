package metrics

// TemperatureStat 表示节点上的关键硬件温度信息（单位：摄氏度）
//
// 示例：
//   {
//     "cpuCoreTemp": 72.5,
//     "gpuTemp": 60.0,
//     "nvmeTemp": 48.3
//   }
type TemperatureStat struct {
	CPUDegrees  float64 `json:"cpuDegrees"`  // CPU 温度（℃）
	GPUDegrees  float64 `json:"gpuDegrees"`  // GPU 温度（可选）
	NVMEDegrees float64 `json:"nvmeDegrees"` // NVMe 温度（可选）
}