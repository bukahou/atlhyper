package collect

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"AtlHyper/model/metrics"
)

var (
	// 采样频率
	cpuInterval = 1 * time.Second // CPU/Load 每 1s 刷新
	topInterval = 3 * time.Second // TopK 每 3s 刷新
)

// ================= 缓存结构 =================

type cpuSnapshot struct {
	Stat    metrics.CPUStat
	Top     []metrics.TopCPUProcess
	Updated time.Time
}

var snapStore atomic.Value // 存 cpuSnapshot

// 进程增量：pid -> (utime+stime)
var (
	lastProcTimes   = map[int]uint64{}
	lastProcTimesMu sync.Mutex
)

// UID->用户名 缓存
var (
	uidCache   = map[string]string{}
	uidCacheMu sync.Mutex
)

// CPU total/idle 跨次采样
var (
	lastTotalCPU     uint64
	lastIdleCPU      uint64
	lastTotalForTop  uint64 // ← Top 归一化所需的系统总 jiffies 基线
)

// ================= 对外函数 =================

// CollectCPU 现在只读缓存
func CollectCPU() (metrics.CPUStat, []metrics.TopCPUProcess, error) {
	v := snapStore.Load()
	if v == nil {
		return metrics.CPUStat{UsagePercent: "N/A", Cores: runtime.NumCPU()}, nil, nil
	}
	s := v.(cpuSnapshot)
	return s.Stat, s.Top, nil
}

// ================= 后台自动采集 =================

func init() {
	go samplerLoop()
}

func samplerLoop() {
	cpuTicker := time.NewTicker(cpuInterval)
	topTicker := time.NewTicker(topInterval)
	defer cpuTicker.Stop()
	defer topTicker.Stop()

	var lastTop []metrics.TopCPUProcess

	for {
		select {
		case <-cpuTicker.C:
			// 1) CPU 使用率 + Load
			stat := metrics.CPUStat{}
			nowTotal, nowIdle, err := readTotalCPU()
			if err == nil && lastTotalCPU != 0 && nowTotal > lastTotalCPU {
				td := nowTotal - lastTotalCPU
				id := nowIdle - lastIdleCPU
				ratio := float64(td-id) / float64(td)
				stat.Usage = ratio
				stat.UsagePercent = fmt.Sprintf("%.2f%%", ratio*100)
			} else {
				stat.UsagePercent = "N/A"
			}
			lastTotalCPU, lastIdleCPU = nowTotal, nowIdle
			stat.Cores = runtime.NumCPU()

			// Load
			if b, err := os.ReadFile(filepath.Join(procRoot, "loadavg")); err == nil {
				fields := strings.Fields(string(b))
				if len(fields) >= 3 {
					stat.Load1, _ = strconv.ParseFloat(fields[0], 64)
					stat.Load5, _ = strconv.ParseFloat(fields[1], 64)
					stat.Load15, _ = strconv.ParseFloat(fields[2], 64)
				}
			}

			// 更新缓存（保留旧 Top）
			prev, _ := snapStore.Load().(cpuSnapshot)
			snapStore.Store(cpuSnapshot{
				Stat:    stat,
				Top:     prev.Top,
				Updated: time.Now(),
			})

		case <-topTicker.C:
			// 2) 刷新 TopK
			prev, _ := snapStore.Load().(cpuSnapshot)
			top, err := collectTopKFast(prev.Stat)
			// 本轮算不到（首轮/极短间隔）就沿用上一轮，避免闪烁
			if err != nil || len(top) == 0 {
				top = lastTop
			} else {
				lastTop = top
			}
			snapStore.Store(cpuSnapshot{
				Stat:    prev.Stat,
				Top:     top,
				Updated: time.Now(),
			})
		}
	}
}

// ================= TopK =================

func collectTopKFast(stat metrics.CPUStat) ([]metrics.TopCPUProcess, error) {
	// 读取系统总 jiffies，用于归一化
	nowTotal, _, err := readTotalCPU()
	if err != nil {
		return nil, err
	}
	// 第一次/异常回绕：建立基线并返回空结果，由调用方沿用上一轮
	if lastTotalForTop == 0 || nowTotal <= lastTotalForTop {
		lastTotalForTop = nowTotal
		return []metrics.TopCPUProcess{}, nil
	}
	totalDelta := nowTotal - lastTotalForTop
	lastTotalForTop = nowTotal
	if totalDelta == 0 {
		return []metrics.TopCPUProcess{}, nil
	}

	ents, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, err
	}

	type lite struct {
		pid    int
		comm   string
		deltaJ uint64
	}
	liteList := make([]lite, 0, 256)

	// 轻扫 /proc/[pid]/stat
	lastProcTimesMu.Lock()
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		data, err := os.ReadFile(filepath.Join(procRoot, e.Name(), "stat"))
		if err != nil {
			continue
		}
		f := strings.Fields(string(data))
		if len(f) < 17 {
			continue
		}
		ut, _ := strconv.ParseUint(f[13], 10, 64)
		st, _ := strconv.ParseUint(f[14], 10, 64)
		tot := ut + st

		// ✅ 首次见到该 pid：仅记录基线，不计入本轮
		if last, ok := lastProcTimes[pid]; !ok {
			lastProcTimes[pid] = tot
			continue
		} else {
			// 进程重启/计数回绕
			if tot <= last {
				lastProcTimes[pid] = tot
				continue
			}
			delta := tot - last
			lastProcTimes[pid] = tot

			if delta == 0 {
				continue
			}
			liteList = append(liteList, lite{
				pid:    pid,
				comm:   strings.Trim(f[1], "()"),
				deltaJ: delta,
			})
		}
	}
	lastProcTimesMu.Unlock()

	// 选 TopK
	if len(liteList) == 0 {
		return []metrics.TopCPUProcess{}, nil
	}
	sort.Slice(liteList, func(i, j int) bool { return liteList[i].deltaJ > liteList[j].deltaJ })
	K := 5
	if len(liteList) < K {
		K = len(liteList)
	}
	liteList = liteList[:K]

	top := make([]metrics.TopCPUProcess, 0, K)

	// 富集信息
	for _, it := range liteList {
		var uid, tgid string
		var memKB uint64

		if f, err := os.Open(filepath.Join(procRoot, strconv.Itoa(it.pid), "status")); err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "Uid:") {
					uid = strings.Fields(line)[1]
				} else if strings.HasPrefix(line, "VmRSS:") {
					memKB, _ = strconv.ParseUint(strings.Fields(line)[1], 10, 64)
				} else if strings.HasPrefix(line, "Tgid:") {
					tgid = strings.Fields(line)[1]
				}
			}
			f.Close()
		}

		// 仅保留进程（过滤线程）
		if tgid != "" && tgid != strconv.Itoa(it.pid) {
			continue
		}

		// 解析用户名（带缓存）
		username := uid
		uidCacheMu.Lock()
		if name, ok := uidCache[uid]; ok {
			username = name
			uidCacheMu.Unlock()
		} else {
			uidCacheMu.Unlock()
			if u, err := user.LookupId(uid); err == nil {
				username = u.Username
				uidCacheMu.Lock()
				uidCache[uid] = username
				uidCacheMu.Unlock()
			}
		}

		// ✅ 正确缩放：进程增量 / 系统总增量
		cpuPct := (float64(it.deltaJ) / float64(totalDelta)) * 100.0

		top = append(top, metrics.TopCPUProcess{
			PID:        it.pid,
			User:       username,
			Command:    it.comm,
			CPUPercent: cpuPct,
			MemoryMB:   float64(memKB) / 1024.0,
		})
	}

	sort.Slice(top, func(i, j int) bool { return top[i].CPUPercent > top[j].CPUPercent })
	for i := range top {
		top[i].CPUUsage = fmt.Sprintf("%.2f%%", top[i].CPUPercent)
		top[i].MemoryUsage = fmt.Sprintf("%.2f MB", top[i].MemoryMB)
	}

	return top, nil
}

// ================= /proc/stat 解析 =================

func readTotalCPU() (uint64, uint64, error) {
	f, err := os.Open(filepath.Join(procRoot, "stat"))
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}
		var total, idle uint64
		for i, v := range fields[1:] {
			val, _ := strconv.ParseUint(v, 10, 64)
			total += val
			if i == 3 {
				idle = val
			}
		}
		return total, idle, nil
	}
	return 0, 0, fmt.Errorf("no cpu line found")
}
