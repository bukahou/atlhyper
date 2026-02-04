// Package utils 工具函数
package utils

import "time"

// CPURawStats CPU 原始统计（用于差值计算）
type CPURawStats struct {
	User    uint64 // 用户态时间
	Nice    uint64 // Nice 用户态时间
	System  uint64 // 内核态时间
	Idle    uint64 // 空闲时间
	IOWait  uint64 // I/O 等待时间
	IRQ     uint64 // 硬中断时间
	SoftIRQ uint64 // 软中断时间
	Steal   uint64 // Steal 时间（虚拟化环境）
}

// Total 返回总时间
func (s CPURawStats) Total() uint64 {
	return s.User + s.Nice + s.System + s.Idle + s.IOWait + s.IRQ + s.SoftIRQ + s.Steal
}

// CPURawSample CPU 采样数据
type CPURawSample struct {
	Timestamp time.Time
	Total     CPURawStats   // 总计
	PerCore   []CPURawStats // 每核
}

// DiskRawStats 磁盘原始统计（用于差值计算）
type DiskRawStats struct {
	Device       string
	ReadComplete uint64 // 读完成次数
	ReadSectors  uint64 // 读扇区数
	WriteComplete uint64 // 写完成次数
	WriteSectors  uint64 // 写扇区数
	IOInProgress uint64 // 正在进行的 I/O
	IOTime       uint64 // I/O 时间（毫秒）
}

// DiskRawSample 磁盘采样数据
type DiskRawSample struct {
	Timestamp time.Time
	Stats     map[string]DiskRawStats // device -> stats
}

// NetRawStats 网络原始统计（用于差值计算）
type NetRawStats struct {
	Interface string
	RxBytes   uint64
	RxPackets uint64
	RxErrors  uint64
	RxDropped uint64
	TxBytes   uint64
	TxPackets uint64
	TxErrors  uint64
	TxDropped uint64
}

// NetRawSample 网络采样数据
type NetRawSample struct {
	Timestamp time.Time
	Stats     map[string]NetRawStats // interface -> stats
}

// ProcRawStats 进程原始统计（用于 CPU 差值计算）
type ProcRawStats struct {
	PID       int
	Name      string
	State     byte
	UTime     uint64 // 用户态时间（jiffies）
	STime     uint64 // 内核态时间（jiffies）
	StartTime uint64 // 启动时间
}

// ProcRawSample 进程采样数据
type ProcRawSample struct {
	Timestamp   time.Time
	Stats       map[int]ProcRawStats // pid -> stats
	TotalCPU    uint64               // 总 CPU 时间（用于计算百分比）
}
