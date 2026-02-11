package metrics

import "strings"

// shouldKeepFilesystem 判断文件系统是否应保留
//
// 规则：只保留 device 以 "/dev/" 开头的条目。
// 过滤掉 shm、tmpfs 等 K8s 容器沙箱产生的噪音。
func shouldKeepFilesystem(device, fstype, mountpoint string) bool {
	return strings.HasPrefix(device, "/dev/")
}

// shouldKeepNetwork 判断网络接口是否应保留
//
// 排除虚拟接口（lo, veth*, flannel*, cni*, cali*），只保留物理接口。
func shouldKeepNetwork(device string) bool {
	switch {
	case device == "lo":
		return false
	case strings.HasPrefix(device, "veth"):
		return false
	case strings.HasPrefix(device, "flannel"):
		return false
	case strings.HasPrefix(device, "cni"):
		return false
	case strings.HasPrefix(device, "cali"):
		return false
	}
	return true
}

// shouldKeepDiskIO 判断磁盘 I/O 设备是否应保留
//
// 排除 dm-*（device-mapper）设备，避免与底层物理设备重复计算。
func shouldKeepDiskIO(device string) bool {
	return !strings.HasPrefix(device, "dm-")
}
