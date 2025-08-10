package collect

import (
	"fmt"
	"syscall"

	"NeuroController/model/metrics"
)

// CollectDisk 采集宿主机的磁盘使用情况（需要宿主机挂载进容器）
func CollectDisk() ([]metrics.DiskStat, error) {
	var result []metrics.DiskStat

	mounts := []struct {
		MountPoint string
		Label      string
	}{
		{"/host_root", "host_root"}, // 宿主机根目录（推荐）
		{"/", "container_root"},     // 容器自身根目录（回退）
	}

	for _, m := range mounts {
		var stat syscall.Statfs_t
		err := syscall.Statfs(m.MountPoint, &stat)
		if err != nil {
			continue
		}

		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bavail * uint64(stat.Bsize)
		used := total - free

		usage := 0.0
		if total > 0 {
			usage = float64(used) / float64(total)
		}

		// 👍 不复用函数，直接本地实现可读转换
		humanReadable := func(b uint64) string {
			gb := float64(b) / (1024 * 1024 * 1024)
			if gb >= 1 {
				return fmt.Sprintf("%.2f GB", gb)
			}
			mb := float64(b) / (1024 * 1024)
			return fmt.Sprintf("%.2f MB", mb)
		}

		result = append(result, metrics.DiskStat{
			MountPoint:    m.Label,
			Total:         total,
			Used:          used,
			Free:          free,
			Usage:         usage,
			TotalReadable: humanReadable(total),
			UsedReadable:  humanReadable(used),
			FreeReadable:  humanReadable(free),
			UsagePercent:  fmt.Sprintf("%.2f%%", usage*100),
		})

		break // 只返回第一个成功挂载的
	}

	return result, nil
}
