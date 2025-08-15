package metrics_store

import (
	"log"
	"os"
	"time"
)

// =======================================================================================
// 📄 metrics_store.go
//
// 🧠 Description:
//     定时从 Agent 拉取最新节点指标快照 (/agent/dataapi/latest)，
//     并将数据写入本地数据库，形成持久化的节点监控数据存档。
//     同时支持通过环境变量 METRICS_SYNC_INTERVAL 配置拉取周期。
//
// 📌 环境变量：
//     METRICS_SYNC_INTERVAL = 拉取周期（time.ParseDuration 格式，例如 "10s"、"1m30s"）
//                            默认值为 30s。
// =======================================================================================

// StartMetricsSync
// ---------------------------------------------------------------------------------------
// 启动节点指标同步的常驻任务：
// 1. 先立即拉取一次数据，避免启动后等待一个周期才有数据。
// 2. 按固定周期（默认 30 秒，可通过 METRICS_SYNC_INTERVAL 配置）拉取 + 入库。
// 3. 内部使用 time.Ticker 实现定时，无需外部传入 ctx。
// ---------------------------------------------------------------------------------------
func StartMetricsSync() {
	// 从环境变量解析拉取间隔，如果未设置或格式错误则回退到默认 30 秒
	interval := parseIntervalFromEnv("METRICS_SYNC_INTERVAL", 15*time.Second)

	// 首次立即执行一次，避免等待
	if err := saveLatestSnapshotsOnce(); err != nil {
		log.Printf("📉 Metrics sync (first run) failed: %v", err)
	} else {
		log.Printf("📈 Metrics sync (first run) OK")
	}

	// 创建周期性定时器
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 循环执行任务
	for range ticker.C {
		if err := saveLatestSnapshotsOnce(); err != nil {
		}
	}
}

// parseIntervalFromEnv
// ---------------------------------------------------------------------------------------
// 从环境变量解析定时任务周期，支持 Go 标准的时间格式：
//   "10s"、"1m"、"1h30m" 等。
// 如果环境变量未设置、格式错误或值 <= 0，则返回默认值。
// ---------------------------------------------------------------------------------------
func parseIntervalFromEnv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
		log.Printf("⚠️ METRICS: invalid %s=%q, fallback to %s", key, v, def)
	}
	return def
}


// ---------------------------------------------------------------------------------------
// 执行一次从 Agent 拉取最新指标数据并写入数据库：
// 1. 调用 master_metrics.GetLatestNodeMetrics() 获取所有节点的最新快照。
// 2. 调用 dbmetrics.UpsertSnapshots() 持久化到数据库（支持 UPSERT 去重/更新）。
// ---------------------------------------------------------------------------------------
// func saveLatestSnapshotsOnce() error {
//     ctx := context.Background()
//     raw, err := master_metrics.GetLatestNodeMetrics()
//     if err != nil {
//         return err
//     }
//     var asArray map[string][]*model.NodeMetricsSnapshot
//     if err := json.Unmarshal(raw, &asArray); err == nil && len(asArray) > 0 {
//         return dbmetrics.UpsertSnapshots(ctx, utils.DB, asArray)
//     }
//     var asObject map[string]*model.NodeMetricsSnapshot
//     if err := json.Unmarshal(raw, &asObject); err == nil && len(asObject) > 0 {
//         arr := make(map[string][]*model.NodeMetricsSnapshot, len(asObject))
//         for node, snap := range asObject {
//             if snap != nil {
//                 arr[node] = []*model.NodeMetricsSnapshot{snap}
//             }
//         }
//         return dbmetrics.UpsertSnapshots(ctx, utils.DB, arr)
//     }

//     return fmt.Errorf("decode /agent/dataapi/latest failed, body=%s",
//         bytes.ReplaceAll(raw, []byte("\n"), []byte{}))
// }
