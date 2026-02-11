//go:build e2e

package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"

	otelpkg "AtlHyper/atlhyper_agent_v2/sdk/impl/otel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_RealOTelData 端到端测试：从真实 OTel Collector 拉取数据
//
// 前提条件：
//   kubectl port-forward -n otel svc/otel-collector 8889:8889
//
// 运行方式：
//   go test ./atlhyper_agent_v2/repository/metrics/ -tags=e2e -v -run TestE2E
func TestE2E_RealOTelData(t *testing.T) {
	otelURL := "http://localhost:8889/metrics"
	healthURL := "http://localhost:13133"

	client := otelpkg.NewOTelClient(otelURL, healthURL, 10*time.Second)

	// 第一次采集
	ctx := context.Background()
	raw1, err := client.ScrapeNodeMetrics(ctx)
	require.NoError(t, err, "第一次 ScrapeNodeMetrics 失败")
	require.NotEmpty(t, raw1, "第一次采集应返回节点数据")

	fmt.Printf("\n=== 第一次采集: %d 个节点 ===\n", len(raw1))
	for name, node := range raw1 {
		fmt.Printf("  %s: instance=%s, arch=%s, cores=%d, mem=%.1fGB, fs=%d, disk=%d, net=%d, temp=%d\n",
			name, node.Instance, node.Machine, node.CPUCoreCount,
			float64(node.MemTotal)/1e9,
			len(node.Filesystems), len(node.DiskIO), len(node.Networks), len(node.HWMonTemps),
		)
	}

	// 验证所有 6 个节点都存在
	expectedNodes := []string{"desk-zero", "desk-one", "desk-two", "raspi-zero", "raspi-one", "raspi-nfs"}
	for _, name := range expectedNodes {
		assert.Contains(t, raw1, name, "缺少节点: %s", name)
	}

	// 验证 desk-zero 详细数据
	if dz, ok := raw1["desk-zero"]; ok {
		assert.Equal(t, "x86_64", dz.Machine)
		assert.Equal(t, 8, dz.CPUCoreCount)
		assert.Greater(t, dz.MemTotal, int64(30e9))       // >30GB
		assert.Greater(t, len(dz.Filesystems), 0)
		assert.Greater(t, len(dz.DiskIO), 0)
		assert.Greater(t, len(dz.Networks), 0)
		assert.Greater(t, len(dz.HWMonTemps), 0)
		assert.Greater(t, dz.TCPCurrEstab, int64(0))
		assert.Greater(t, dz.ConntrackEntries, int64(0))
		assert.Equal(t, float64(1), dz.TimexSyncStatus)
	}

	// 验证 raspi-zero 详细数据
	if rz, ok := raw1["raspi-zero"]; ok {
		assert.Equal(t, "aarch64", rz.Machine)
		assert.Equal(t, 4, rz.CPUCoreCount)
		assert.Greater(t, rz.MemTotal, int64(7e9)) // >7GB
	}

	// 等待 16 秒后做第二次采集（OTel Collector 每 15s 抓取 node_exporter）
	fmt.Println("\n等待 16 秒进行第二次采集（等待 OTel 新数据）...")
	time.Sleep(16 * time.Second)

	raw2, err := client.ScrapeNodeMetrics(ctx)
	require.NoError(t, err, "第二次 ScrapeNodeMetrics 失败")
	require.NotEmpty(t, raw2)

	fmt.Printf("\n=== 第二次采集: %d 个节点 ===\n", len(raw2))

	// 模拟 MetricsRepository 的转换逻辑
	elapsed := 16.0
	fmt.Println("\n=== 转换为 NodeMetricsSnapshot ===")
	for nodeName, cur := range raw2 {
		prev := raw1[nodeName]
		snap := convertToSnapshot(nodeName, cur, prev, elapsed)

		fmt.Printf("\n--- %s ---\n", nodeName)
		fmt.Printf("  CPU: usage=%.1f%%, user=%.1f%%, system=%.1f%%, idle=%.1f%%, iowait=%.1f%%\n",
			snap.CPU.UsagePercent, snap.CPU.UserPercent, snap.CPU.SystemPercent,
			snap.CPU.IdlePercent, snap.CPU.IOWaitPercent)
		fmt.Printf("  CPU: cores=%d, freq=%.0fMHz, load=%.2f/%.2f/%.2f\n",
			snap.CPU.Cores, snap.CPU.Frequency, snap.CPU.Load1, snap.CPU.Load5, snap.CPU.Load15)
		fmt.Printf("  Memory: used=%.1fGB/%.1fGB (%.1f%%)\n",
			float64(snap.Memory.Used)/1e9, float64(snap.Memory.Total)/1e9, snap.Memory.UsagePercent)
		for _, d := range snap.Disks {
			fmt.Printf("  Disk %s [%s]: %.1fGB/%.1fGB (%.1f%%), R=%.0fKB/s W=%.0fKB/s IOUtil=%.1f%%\n",
				d.Device, d.MountPoint,
				float64(d.Used)/1e9, float64(d.Total)/1e9, d.UsagePercent,
				d.ReadRate/1024, d.WriteRate/1024, d.IOUtil)
		}
		for _, n := range snap.Networks {
			fmt.Printf("  Net %s [%s]: Rx=%.1fKB/s Tx=%.1fKB/s\n",
				n.Interface, n.Status, n.RxRate/1024, n.TxRate/1024)
		}
		fmt.Printf("  Temp: CPU=%.1f°C (max=%.1f°C)\n", snap.Temperature.CPUTemp, snap.Temperature.CPUTempMax)
		fmt.Printf("  PSI: cpu=%.2f%%, io_some=%.2f%%, io_full=%.2f%%\n",
			snap.PSI.CPUSomePercent, snap.PSI.IOSomePercent, snap.PSI.IOFullPercent)
		fmt.Printf("  TCP: estab=%d, tw=%d, alloc=%d, inuse=%d\n",
			snap.TCP.CurrEstab, snap.TCP.TimeWait, snap.TCP.Alloc, snap.TCP.InUse)
		fmt.Printf("  System: conntrack=%d/%d, filefd=%d, entropy=%d\n",
			snap.System.ConntrackEntries, snap.System.ConntrackLimit,
			snap.System.FilefdAllocated, snap.System.EntropyAvailable)
		fmt.Printf("  NTP: synced=%v, offset=%.6fs\n", snap.NTP.Synced, snap.NTP.OffsetSeconds)
		fmt.Printf("  Softnet: dropped=%d, squeezed=%d\n", snap.Softnet.Dropped, snap.Softnet.Squeezed)

		// 验证转换结果合理性
		assert.GreaterOrEqual(t, snap.CPU.UsagePercent, 0.0)
		assert.LessOrEqual(t, snap.CPU.UsagePercent, 100.0)
		assert.Greater(t, snap.Memory.Total, int64(0))
		assert.GreaterOrEqual(t, snap.Memory.UsagePercent, 0.0)
		assert.Greater(t, len(snap.Disks), 0)
		assert.Greater(t, len(snap.Networks), 0)
		assert.True(t, snap.NTP.Synced, "NTP 应该已同步")
	}
}
