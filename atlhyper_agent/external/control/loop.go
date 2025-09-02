package control

import (
	"context"
	"log"
	"time"

	"AtlHyper/atlhyper_agent/interfaces/operations"
)

// StartControlLoop —— 控制循环：持续从 Master 拉取并执行值令
// 参数：
//   - clusterID   集群唯一标识
//   - opsBasePath Master 的操作下发接口路径，例如 "/ingest/ops"
func StartControlLoop(clusterID, opsBasePath string) {
	client := NewClient(opsBasePath, clusterID, 30)

	go func() {
		var lastRV uint64 = 0
		backoff := 1 * time.Second
		const backoffMax = 30 * time.Second

		for {
			ctx := context.Background()

			// 1) 长轮询 /watch
			set, changed, err := client.Watch(ctx, lastRV)
			if err != nil {
				log.Printf("[控制循环] 监视出错: %v (回退等待 %s)", err, backoff)
				time.Sleep(backoff)
				if backoff < backoffMax {
					backoff *= 2
				}
				continue
			}
			if !changed {
				// 无更新：下一轮
				backoff = 1 * time.Second
				continue
			}
			backoff = 1 * time.Second

			// ✅ 若是空集，不执行也不 ACK，只推进本地 RV，避免空转
			if len(set.Commands) == 0 {
				lastRV = set.RV
				continue
			}

			log.Printf("[控制循环] 收到新命令: rv=%d 共 %d 条", set.RV, len(set.Commands))

			// 2) 执行命令 → 调用接口层 Execute
			results := make([]AckResult, 0, len(set.Commands))
			for _, cmd := range set.Commands {
				start := time.Now()

				// 执行前打印
				log.Printf("[控制循环] 开始执行: 类型=%s 目标=%v 幂等键=%s",
					cmd.Type, cmd.Target, cmd.Idem)

				r := operations.Execute(ctx, operations.Command{
					ID:     cmd.ID,
					Type:   cmd.Type,
					Target: cmd.Target,
					Args:   cmd.Args,
					Idem:   cmd.Idem,
				})

				// 执行后打印
				log.Printf("[控制循环] 执行完成: 命令ID=%s 状态=%s 信息=%s",
					r.CommandID, r.Status, r.Message)

				results = append(results, AckResult{
					CommandID:  r.CommandID,
					Status:     r.Status,
					Message:    r.Message,
					ErrorCode:  r.ErrorCode,
					StartedAt:  start.Format(time.RFC3339),
					FinishedAt: time.Now().Format(time.RFC3339),
				})
			}

			// 3) 回执 /ack（成功后推进 RV）
			if err := client.Ack(ctx, results); err != nil {
				log.Printf("[控制循环] 回执失败: %v (将在下轮重试)", err)
				// 不更新 RV，下一轮仍会拿到相同命令；幂等由 Execute 保障
				continue
			}
			lastRV = set.RV
			log.Printf("[控制循环] 回执成功: 已确认 rv=%d", lastRV)
		}
	}()
}
