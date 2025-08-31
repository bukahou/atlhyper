package metrics

import (
	"context"
	"database/sql"
	"time"

	model "AtlHyper/model/metrics"
)

// upsertSnapshots 将 map[node][]*snapshot 批量写入两张表（事务）
func UpsertSnapshots(ctx context.Context, db *sql.DB, data map[string][]*model.NodeMetricsSnapshot) (err error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 主表 UPSERT
	const upsertMain = `
INSERT INTO node_metrics_flat
(node_name, ts,
 cpu_usage, cpu_cores, cpu_load1, cpu_load5, cpu_load15,
 memory_total, memory_used, memory_available, memory_usage,
 temp_cpu, temp_gpu, temp_nvme,
 disk_total, disk_used, disk_free, disk_usage,
 net_lo_rx_kbps, net_lo_tx_kbps, net_eth0_rx_kbps, net_eth0_tx_kbps)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(node_name, ts) DO UPDATE SET
 cpu_usage=excluded.cpu_usage, cpu_cores=excluded.cpu_cores,
 cpu_load1=excluded.cpu_load1, cpu_load5=excluded.cpu_load5, cpu_load15=excluded.cpu_load15,
 memory_total=excluded.memory_total, memory_used=excluded.memory_used,
 memory_available=excluded.memory_available, memory_usage=excluded.memory_usage,
 temp_cpu=excluded.temp_cpu, temp_gpu=excluded.temp_gpu, temp_nvme=excluded.temp_nvme,
 disk_total=excluded.disk_total, disk_used=excluded.disk_used, disk_free=excluded.disk_free, disk_usage=excluded.disk_usage,
 net_lo_rx_kbps=excluded.net_lo_rx_kbps, net_lo_tx_kbps=excluded.net_lo_tx_kbps,
 net_eth0_rx_kbps=excluded.net_eth0_rx_kbps, net_eth0_tx_kbps=excluded.net_eth0_tx_kbps;
`
	stmtMain, err := tx.PrepareContext(ctx, upsertMain)
	if err != nil {
		return err
	}
	defer stmtMain.Close()

	// 进程表 UPSERT
	const upsertProc = `
INSERT INTO node_top_processes
(node_name, ts, pid, user, command, cpu_percent, memory_mb)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(node_name, ts, pid) DO UPDATE SET
 user=excluded.user, command=excluded.command,
 cpu_percent=excluded.cpu_percent, memory_mb=excluded.memory_mb;
`
	stmtProc, err := tx.PrepareContext(ctx, upsertProc)
	if err != nil {
		return err
	}
	defer stmtProc.Close()

	for _, arr := range data { // 不使用 node，改成 _
		if len(arr) == 0 {
			continue
		}
		// /latest 一般只有 1 条；为稳妥取最后一条
		s := arr[len(arr)-1]

		// 时间处理
		var ts string
		switch t := any(s.Timestamp).(type) {
		case time.Time:
			ts = t.UTC().Format(time.RFC3339Nano)
		case string:
			ts = t
		default:
			continue
		}

		// 磁盘：优先 host_root，否则第一个
		var dTotal, dUsed, dFree int64
		var dUsage float64
		if len(s.Disk) > 0 {
			idx := 0
			for i, d := range s.Disk {
				if d.MountPoint == "host_root" {
					idx = i
					break
				}
			}
			dTotal = toInt64(s.Disk[idx].Total)
			dUsed = toInt64(s.Disk[idx].Used)
			dFree  = toInt64(s.Disk[idx].Free)
			dUsage = s.Disk[idx].Usage
		}

		// 网络：只取 lo / eth0
		var loRX, loTX, ethRX, ethTX float64
		for _, n := range s.Network {
			switch n.Interface {
			case "lo":
				loRX = n.RxKBps
				loTX = n.TxKBps
			case "eth0":
				ethRX = n.RxKBps
				ethTX = n.TxKBps
			}
		}

		// 主表写入
		if _, err = stmtMain.ExecContext(
			ctx,
			s.NodeName, ts,
			s.CPU.Usage, s.CPU.Cores, s.CPU.Load1, s.CPU.Load5, s.CPU.Load15,
			toInt64(s.Memory.Total), toInt64(s.Memory.Used), toInt64(s.Memory.Available), s.Memory.Usage,
			int64(s.Temperature.CPUDegrees), int64(s.Temperature.GPUDegrees), int64(s.Temperature.NVMEDegrees),
			dTotal, dUsed, dFree, dUsage,
			loRX, loTX, ethRX, ethTX,
		); err != nil {
			return err
		}

		// 进程表写入
		for _, p := range s.TopCPUProcesses {
			if _, err = stmtProc.ExecContext(
				ctx,
				s.NodeName, ts,
				int64(p.PID), p.User, p.Command, p.CPUPercent, p.MemoryMB, // 改成 p.PID
			); err != nil {
				return err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// 小工具：兼容到 int64
func toInt64[T ~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64 | ~float64](v T) int64 {
	return int64(v)
}


func CleanupOldSnapshots(ctx context.Context, db *sql.DB, retention time.Duration) (int64, int64, error) {
	cutoff := time.Now().Add(-retention).UTC().Format(time.RFC3339Nano)

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res1, err := tx.ExecContext(ctx, `DELETE FROM node_metrics_flat WHERE ts < ?;`, cutoff)
	if err != nil {
		return 0, 0, err
	}
	aff1, _ := res1.RowsAffected()

	res2, err := tx.ExecContext(ctx, `DELETE FROM node_top_processes WHERE ts < ?;`, cutoff)
	if err != nil {
		return 0, 0, err
	}
	aff2, _ := res2.RowsAffected()

	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	return aff1, aff2, nil
}

// SaveLatestSnapshots 拉取 /agent/dataapi/latest 并入库
// func SaveLatestSnapshots(ctx context.Context) error {
// 	data, err := master_metrics.GetLatestNodeMetrics()
// 	if err != nil {
// 		return fmt.Errorf("pull latest from agent: %w", err)
// 	}
// 	return upsertSnapshots(ctx, utils.DB, data)
// }