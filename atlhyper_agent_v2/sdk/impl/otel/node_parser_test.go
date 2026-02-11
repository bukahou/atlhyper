package otel

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

func loadTestData(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile("../../../testdata/" + filename)
	require.NoError(t, err, "failed to load testdata/%s", filename)
	return string(data)
}

// findFS 从文件系统列表中按挂载点查找
func findFS(fsList []sdk.FSRawMetrics, mountpoint string) *sdk.FSRawMetrics {
	for i := range fsList {
		if fsList[i].MountPoint == mountpoint {
			return &fsList[i]
		}
	}
	return nil
}

// findTemp 从温度列表中按芯片和传感器查找
func findTemp(temps []sdk.HWMonRawTemp, chip, sensor string) *sdk.HWMonRawTemp {
	for i := range temps {
		if temps[i].Chip == chip && temps[i].Sensor == sensor {
			return &temps[i]
		}
	}
	return nil
}

func Test_ParseNodeMetrics_DeskZero(t *testing.T) {
	text := loadTestData(t, "otel_desk_zero.txt")
	result := parseNodeMetrics(text)

	require.Contains(t, result, "desk-zero")
	node := result["desk-zero"]

	// 基础信息
	assert.Equal(t, "desk-zero", node.NodeName)
	assert.Equal(t, "192.168.0.130:9100", node.Instance)
	assert.Equal(t, "x86_64", node.Machine)
	assert.Equal(t, "6.8.0-85-generic", node.Kernel)
	assert.InDelta(t, 1.759572263e+09, node.BootTime, 1)

	// CPU
	assert.Equal(t, 8, node.CPUCoreCount)
	assert.Len(t, node.CPUSecondsTotal, 64) // 8核 × 8模式
	assert.InDelta(t, 1.066988737e+07, node.CPUSecondsTotal["0:idle"], 1)
	assert.InDelta(t, 325095.17, node.CPUSecondsTotal["0:user"], 1)

	// Load
	assert.InDelta(t, 0.56, node.Load1, 0.01)
	assert.InDelta(t, 0.41, node.Load5, 0.01)
	assert.InDelta(t, 0.28, node.Load15, 0.01)

	// CPU 频率
	assert.Greater(t, len(node.CPUFreqHertz), 0)

	// Memory
	assert.Equal(t, int64(33528840192), node.MemTotal)
	assert.Equal(t, int64(29597294592), node.MemAvailable)
	assert.Equal(t, int64(18630926336), node.MemFree)
	assert.Equal(t, int64(9811787776), node.MemCached)
	assert.Equal(t, int64(961503232), node.MemBuffers)
	assert.Equal(t, int64(0), node.SwapTotal)
	assert.Equal(t, int64(0), node.SwapFree)

	// Filesystem (过滤后: /, /boot, /boot/efi)
	assert.Len(t, node.Filesystems, 3)
	rootFS := findFS(node.Filesystems, "/")
	require.NotNil(t, rootFS)
	assert.Equal(t, "/dev/mapper/ubuntu--vg-ubuntu--lv", rootFS.Device)
	assert.Equal(t, int64(105089261568), rootFS.SizeBytes)
	assert.Equal(t, int64(87484624896), rootFS.AvailBytes)

	// Disk I/O (过滤掉 dm-0, 只有 sda)
	assert.Len(t, node.DiskIO, 1)
	assert.Equal(t, "sda", node.DiskIO[0].Device)
	assert.InDelta(t, 4.22935286272e+11, node.DiskIO[0].ReadBytesTotal, 1)

	// Network (过滤后: 只有 eno1)
	assert.Len(t, node.Networks, 1)
	assert.Equal(t, "eno1", node.Networks[0].Device)
	assert.True(t, node.Networks[0].Up)
	assert.Equal(t, int64(125000000), node.Networks[0].Speed) // 1Gbps

	// Temperature
	assert.GreaterOrEqual(t, len(node.HWMonTemps), 5) // coretemp 5 sensors
	cpuTemp := findTemp(node.HWMonTemps, "platform_coretemp_0", "temp1")
	require.NotNil(t, cpuTemp)
	assert.InDelta(t, 49.0, cpuTemp.Current, 1)
	assert.InDelta(t, 74.0, cpuTemp.Max, 1)

	// PSI
	assert.InDelta(t, 110989.082491, node.PSICPUWaiting, 1)

	// TCP
	assert.Equal(t, int64(138), node.TCPCurrEstab)
	assert.Equal(t, int64(49), node.TCPTimeWait)

	// System
	assert.Equal(t, int64(262144), node.ConntrackLimit)
	assert.Equal(t, int64(3840), node.FilefdAllocated)
	assert.Equal(t, int64(256), node.EntropyBits)

	// NTP
	assert.Equal(t, float64(1), node.TimexSyncStatus)

	// Softnet (所有 CPU 求和)
	assert.Equal(t, int64(0), node.SoftnetDropped)
	assert.GreaterOrEqual(t, node.SoftnetSqueezed, int64(200)) // ~204
}

func Test_ParseNodeMetrics_RaspiZero(t *testing.T) {
	text := loadTestData(t, "otel_raspi_zero.txt")
	result := parseNodeMetrics(text)

	require.Contains(t, result, "raspi-zero")
	node := result["raspi-zero"]

	assert.Equal(t, "aarch64", node.Machine)
	assert.Equal(t, 4, node.CPUCoreCount)
	assert.Len(t, node.CPUSecondsTotal, 32) // 4核 × 8模式

	// Filesystem
	assert.Len(t, node.Filesystems, 2) // /, /boot/firmware

	// Network
	assert.Len(t, node.Networks, 2) // eth0 (up), wlan0 (down)

	// Temperature — arm64 芯片名不同
	assert.Greater(t, len(node.HWMonTemps), 0)

	// ConntrackLimit 不同
	assert.Equal(t, int64(131072), node.ConntrackLimit) // raspi 上限更低
}

func Test_ParseNodeMetrics_MultipleNodes(t *testing.T) {
	text := loadTestData(t, "otel_all_nodes.txt")
	result := parseNodeMetrics(text)

	assert.Len(t, result, 6)
	for _, name := range []string{"desk-zero", "desk-one", "desk-two", "raspi-zero", "raspi-one", "raspi-nfs"} {
		assert.Contains(t, result, name)
		assert.NotEmpty(t, result[name].NodeName)
		assert.NotEmpty(t, result[name].Instance)
		assert.Greater(t, result[name].CPUCoreCount, 0)
		assert.Greater(t, result[name].MemTotal, int64(0))
	}
}

func Test_ParseNodeMetrics_EmptyInput(t *testing.T) {
	result := parseNodeMetrics("")
	assert.Empty(t, result)
}

func Test_ParseNodeMetrics_NoNodeMetrics(t *testing.T) {
	// 只有 SLO 指标，无 node_exporter 指标
	result := parseNodeMetrics("otel_response_total{pod=\"test\"} 100\n")
	assert.Empty(t, result)
}
