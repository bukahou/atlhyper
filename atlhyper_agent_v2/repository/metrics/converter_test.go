package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

func Test_ConvertToSnapshot_CPUUsage(t *testing.T) {
	prev := &sdk.OTelNodeRawMetrics{
		CPUSecondsTotal: map[string]float64{
			"0:idle": 1000, "0:user": 100, "0:system": 50,
			"0:iowait": 5, "0:nice": 0, "0:irq": 0, "0:softirq": 2, "0:steal": 0,
		},
		CPUCoreCount: 1,
	}
	cur := &sdk.OTelNodeRawMetrics{
		CPUSecondsTotal: map[string]float64{
			"0:idle": 1010, "0:user": 103, "0:system": 51.5,
			"0:iowait": 5.2, "0:nice": 0, "0:irq": 0, "0:softirq": 2.3, "0:steal": 0,
		},
		CPUCoreCount:    1,
		Load1:           0.5,
		Load5:           0.4,
		Load15:          0.3,
		CPUFreqHertz:    map[string]float64{"0": 2.4e9},
		CPUFreqMaxHertz: 3.8e9,
	}
	snap := convertToSnapshot("test", cur, prev, 15.0)

	// 总 delta = (1010-1000)+(103-100)+(51.5-50)+(5.2-5)+(2.3-2) = 10+3+1.5+0.2+0.3 = 15
	// idle delta = 10
	// usage = (15-10)/15 * 100 = 33.33%
	assert.InDelta(t, 33.33, snap.CPU.UsagePercent, 0.1)
	assert.InDelta(t, 20.0, snap.CPU.UserPercent, 0.1)  // 3/15*100
	assert.InDelta(t, 10.0, snap.CPU.SystemPercent, 0.1) // 1.5/15*100
	assert.InDelta(t, 66.67, snap.CPU.IdlePercent, 0.1)  // 10/15*100
	assert.InDelta(t, 1.33, snap.CPU.IOWaitPercent, 0.1) // 0.2/15*100
	assert.InDelta(t, 0.5, snap.CPU.Load1, 0.01)
	assert.Equal(t, 1, snap.CPU.Cores)
	assert.Equal(t, 1, snap.CPU.Threads)
	assert.InDelta(t, 2400, snap.CPU.Frequency, 1) // Hz → MHz
}

func Test_ConvertToSnapshot_Memory(t *testing.T) {
	cur := &sdk.OTelNodeRawMetrics{
		MemTotal:        33528840192,
		MemAvailable:    29597294592,
		MemFree:         18630926336,
		MemCached:       9811787776,
		MemBuffers:      961503232,
		SwapTotal:       0,
		SwapFree:        0,
		CPUSecondsTotal: make(map[string]float64),
		CPUFreqHertz:    make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, nil, 15.0)

	assert.Equal(t, int64(33528840192), snap.Memory.Total)
	assert.Equal(t, int64(29597294592), snap.Memory.Available)
	used := int64(33528840192 - 29597294592) // Total - Available
	assert.Equal(t, used, snap.Memory.Used)
	percent := float64(used) / float64(33528840192) * 100
	assert.InDelta(t, percent, snap.Memory.UsagePercent, 0.1) // ~11.7%
	assert.Equal(t, int64(0), snap.Memory.SwapUsed)
	assert.InDelta(t, 0.0, snap.Memory.SwapPercent, 0.1)
}

func Test_ConvertToSnapshot_DiskRate(t *testing.T) {
	prev := &sdk.OTelNodeRawMetrics{
		DiskIO: []sdk.DiskIORawMetrics{{
			Device:               "sda",
			ReadBytesTotal:       1000000,
			WrittenBytesTotal:    2000000,
			ReadsCompletedTotal:  100,
			WritesCompletedTotal: 200,
			IOTimeSecondsTotal:   10.0,
		}},
		CPUSecondsTotal: make(map[string]float64),
		CPUFreqHertz:    make(map[string]float64),
	}
	cur := &sdk.OTelNodeRawMetrics{
		DiskIO: []sdk.DiskIORawMetrics{{
			Device:               "sda",
			ReadBytesTotal:       1150000,
			WrittenBytesTotal:    2300000,
			ReadsCompletedTotal:  110,
			WritesCompletedTotal: 220,
			IOTimeSecondsTotal:   12.5,
		}},
		Filesystems: []sdk.FSRawMetrics{{
			Device: "/dev/sda1", MountPoint: "/", FSType: "ext4",
			SizeBytes: 100e9, AvailBytes: 60e9,
		}},
		CPUSecondsTotal: make(map[string]float64),
		CPUFreqHertz:    make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, prev, 15.0)

	assert.Len(t, snap.Disks, 1)
	assert.InDelta(t, 10000, snap.Disks[0].ReadRate, 1)     // 150000/15
	assert.InDelta(t, 20000, snap.Disks[0].WriteRate, 1)     // 300000/15
	assert.InDelta(t, 0.667, snap.Disks[0].ReadIOPS, 0.01)   // 10/15
	assert.InDelta(t, 1.333, snap.Disks[0].WriteIOPS, 0.01)  // 20/15
	// IO util = delta(io_time) / elapsed * 100 = 2.5/15*100 = 16.67%
	assert.InDelta(t, 16.67, snap.Disks[0].IOUtil, 0.1)
}

func Test_ConvertToSnapshot_PSI(t *testing.T) {
	prev := &sdk.OTelNodeRawMetrics{
		PSICPUWaiting:    100.0,
		PSIMemoryWaiting: 0.5,
		PSIMemoryStalled: 0.1,
		PSIIOWaiting:     10.0,
		PSIIOStalled:     5.0,
		CPUSecondsTotal:  make(map[string]float64),
		CPUFreqHertz:     make(map[string]float64),
	}
	cur := &sdk.OTelNodeRawMetrics{
		PSICPUWaiting:    101.5, // +1.5s in 15s
		PSIMemoryWaiting: 0.5,   // no change
		PSIMemoryStalled: 0.1,
		PSIIOWaiting:     10.75, // +0.75s in 15s
		PSIIOStalled:     5.3,   // +0.3s in 15s
		CPUSecondsTotal:  make(map[string]float64),
		CPUFreqHertz:     make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, prev, 15.0)

	// PSI CPU some = 1.5/15 * 100 = 10.0%
	assert.InDelta(t, 10.0, snap.PSI.CPUSomePercent, 0.1)
	// PSI Memory some = 0/15 * 100 = 0%
	assert.InDelta(t, 0.0, snap.PSI.MemorySomePercent, 0.1)
	// PSI IO some = 0.75/15 * 100 = 5.0%
	assert.InDelta(t, 5.0, snap.PSI.IOSomePercent, 0.1)
	// PSI IO full = 0.3/15 * 100 = 2.0%
	assert.InDelta(t, 2.0, snap.PSI.IOFullPercent, 0.1)
}

func Test_ConvertToSnapshot_NoPrev(t *testing.T) {
	cur := &sdk.OTelNodeRawMetrics{
		CPUSecondsTotal: map[string]float64{
			"0:idle": 1010, "0:user": 103, "0:system": 51.5,
			"0:iowait": 5.2, "0:nice": 0, "0:irq": 0, "0:softirq": 2.3, "0:steal": 0,
		},
		CPUCoreCount:    1,
		Load1:           0.5,
		Load5:           0.4,
		Load15:          0.3,
		MemTotal:        33528840192,
		MemAvailable:    29597294592,
		CPUFreqHertz:    make(map[string]float64),
		CPUFreqMaxHertz: 3.8e9,
	}
	snap := convertToSnapshot("test", cur, nil, 15.0)

	assert.InDelta(t, 0, snap.CPU.UsagePercent, 0.01) // 无 prev 无法算 rate
	assert.InDelta(t, 0.5, snap.CPU.Load1, 0.01)      // gauge 正常
	assert.Equal(t, int64(33528840192), snap.Memory.Total)
	assert.InDelta(t, 0, snap.PSI.CPUSomePercent, 0.01)
}

func Test_ConvertToSnapshot_Temperature_AMD64(t *testing.T) {
	cur := &sdk.OTelNodeRawMetrics{
		HWMonTemps: []sdk.HWMonRawTemp{
			{Chip: "platform_coretemp_0", Sensor: "temp1", Current: 53, Max: 74, Critical: 80},
			{Chip: "platform_coretemp_0", Sensor: "temp2", Current: 49, Max: 74, Critical: 80},
		},
		CPUSecondsTotal: make(map[string]float64),
		CPUFreqHertz:    make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, nil, 15.0)

	assert.InDelta(t, 53.0, snap.Temperature.CPUTemp, 0.1)    // 取 Package (temp1)
	assert.InDelta(t, 74.0, snap.Temperature.CPUTempMax, 0.1)
	assert.Len(t, snap.Temperature.Sensors, 2)
}

func Test_ConvertToSnapshot_Temperature_ARM64(t *testing.T) {
	cur := &sdk.OTelNodeRawMetrics{
		HWMonTemps: []sdk.HWMonRawTemp{
			{Chip: "1000120000_pcie_1f000c8000_adc", Sensor: "temp1", Current: 56.05},
			{Chip: "thermal_thermal_zone0", Sensor: "temp0", Current: 53.45},
			{Chip: "nvme_nvme0", Sensor: "temp1", Current: 34.85, Max: 81.85},
		},
		CPUSecondsTotal: make(map[string]float64),
		CPUFreqHertz:    make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, nil, 15.0)

	// arm64: 取 thermal_zone 或 adc 的最高值作为 CPU 温度
	assert.InDelta(t, 56.05, snap.Temperature.CPUTemp, 0.1)
}

func Test_ConvertToSnapshot_Softnet(t *testing.T) {
	cur := &sdk.OTelNodeRawMetrics{
		SoftnetDropped:  0,
		SoftnetSqueezed: 204,
		CPUSecondsTotal: make(map[string]float64),
		CPUFreqHertz:    make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, nil, 15.0)
	assert.Equal(t, int64(0), snap.Softnet.Dropped)
	assert.Equal(t, int64(204), snap.Softnet.Squeezed)
}

func Test_ConvertToSnapshot_TCPAndSystem(t *testing.T) {
	cur := &sdk.OTelNodeRawMetrics{
		TCPCurrEstab:     138,
		TCPTimeWait:      49,
		TCPOrphan:        0,
		TCPAlloc:         468,
		TCPInUse:         69,
		SocketsUsed:      782,
		ConntrackEntries: 7582,
		ConntrackLimit:   262144,
		FilefdAllocated:  3840,
		FilefdMaximum:    int64(9223372036854775807),
		EntropyBits:      256,
		CPUSecondsTotal:  make(map[string]float64),
		CPUFreqHertz:     make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, nil, 15.0)

	assert.Equal(t, int64(138), snap.TCP.CurrEstab)
	assert.Equal(t, int64(49), snap.TCP.TimeWait)
	assert.Equal(t, int64(0), snap.TCP.Orphan)
	assert.Equal(t, int64(468), snap.TCP.Alloc)
	assert.Equal(t, int64(69), snap.TCP.InUse)
	assert.Equal(t, int64(782), snap.TCP.SocketsUsed)

	assert.Equal(t, int64(7582), snap.System.ConntrackEntries)
	assert.Equal(t, int64(262144), snap.System.ConntrackLimit)
	assert.Equal(t, int64(3840), snap.System.FilefdAllocated)
	assert.Equal(t, int64(256), snap.System.EntropyAvailable)
}

func Test_ConvertToSnapshot_NTP(t *testing.T) {
	cur := &sdk.OTelNodeRawMetrics{
		TimexOffsetSeconds: 0.000168911,
		TimexSyncStatus:    1,
		CPUSecondsTotal:    make(map[string]float64),
		CPUFreqHertz:       make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, nil, 15.0)

	assert.InDelta(t, 0.000168911, snap.NTP.OffsetSeconds, 1e-8)
	assert.True(t, snap.NTP.Synced)
}

func Test_ConvertToSnapshot_VMStat(t *testing.T) {
	prev := &sdk.OTelNodeRawMetrics{
		PgFault:         1000000,
		PgMajFault:      100,
		PswpIn:          0,
		PswpOut:         0,
		CPUSecondsTotal: make(map[string]float64),
		CPUFreqHertz:    make(map[string]float64),
	}
	cur := &sdk.OTelNodeRawMetrics{
		PgFault:         1001500,
		PgMajFault:      103,
		PswpIn:          0,
		PswpOut:         0,
		CPUSecondsTotal: make(map[string]float64),
		CPUFreqHertz:    make(map[string]float64),
	}
	snap := convertToSnapshot("test", cur, prev, 15.0)

	assert.InDelta(t, 100.0, snap.VMStat.PgFaultPS, 0.1)   // 1500/15
	assert.InDelta(t, 0.2, snap.VMStat.PgMajFaultPS, 0.01) // 3/15
	assert.InDelta(t, 0.0, snap.VMStat.PswpInPS, 0.01)
	assert.InDelta(t, 0.0, snap.VMStat.PswpOutPS, 0.01)
}
