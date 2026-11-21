package metrics

import "testing"

func TestPrimaryDiskIndex(t *testing.T) {
	cases := []struct {
		name  string
		disks []NodeDisk
		want  int
	}{
		{"根分区优先，即使更小", []NodeDisk{
			{MountPoint: "/data", TotalBytes: 4 << 40},
			{MountPoint: "/", TotalBytes: 64 << 30},
		}, 1},
		{"没有根分区取容量最大的已挂载分区", []NodeDisk{
			{MountPoint: "/boot/firmware", TotalBytes: 512 << 20},
			{MountPoint: "/var/lib", TotalBytes: 256 << 30},
		}, 1},
		{"没有挂载点的块设备不参与", []NodeDisk{
			{Device: "sda", TotalBytes: 8 << 40},
			{MountPoint: "/srv", TotalBytes: 32 << 30},
		}, 1},
		{"全都没有挂载点", []NodeDisk{{Device: "sda"}}, -1},
		{"没有盘", nil, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := &NodeMetrics{Disks: c.disks}
			if got := m.PrimaryDiskIndex(); got != c.want {
				t.Errorf("PrimaryDiskIndex() = %d, want %d", got, c.want)
			}
			if d := m.GetPrimaryDisk(); (d == nil) != (c.want < 0) {
				t.Errorf("GetPrimaryDisk() = %v，与下标 %d 不一致", d, c.want)
			}
		})
	}
}
