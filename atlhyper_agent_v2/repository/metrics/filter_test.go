package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ShouldKeepFilesystem(t *testing.T) {
	tests := []struct {
		name       string
		device     string
		fstype     string
		mountpoint string
		want       bool
	}{
		{"root ext4", "/dev/mapper/ubuntu--vg-ubuntu--lv", "ext4", "/", true},
		{"boot ext4", "/dev/sda2", "ext4", "/boot", true},
		{"efi vfat", "/dev/sda1", "vfat", "/boot/efi", true},
		{"nvme root", "/dev/nvme0n1p2", "ext4", "/", true},
		{"shm tmpfs", "shm", "tmpfs", "/run/k3s/containerd/xxx/shm", false},
		{"tmpfs run", "tmpfs", "tmpfs", "/run", false},
		{"proc", "proc", "proc", "/proc", false},
		{"sysfs", "sysfs", "sysfs", "/sys", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldKeepFilesystem(tt.device, tt.fstype, tt.mountpoint))
		})
	}
}

func Test_ShouldKeepNetwork(t *testing.T) {
	tests := []struct {
		name   string
		device string
		want   bool
	}{
		{"physical eno1", "eno1", true},
		{"physical eth0", "eth0", true},
		{"wifi wlan0", "wlan0", true},
		{"loopback", "lo", false},
		{"veth pair", "veth12345abc", false},
		{"flannel", "flannel.1", false},
		{"cni bridge", "cni0", false},
		{"calico", "cali12345", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldKeepNetwork(tt.device))
		})
	}
}

func Test_ShouldKeepDiskIO(t *testing.T) {
	tests := []struct {
		name   string
		device string
		want   bool
	}{
		{"physical sda", "sda", true},
		{"nvme", "nvme0n1", true},
		{"device-mapper 0", "dm-0", false},
		{"device-mapper 1", "dm-1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldKeepDiskIO(tt.device))
		})
	}
}
