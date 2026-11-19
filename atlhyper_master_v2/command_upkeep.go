package atlhyper_master_v2

import (
	"AtlHyper/atlhyper_master_v2/datahub"
	"context"
	"time"
)

// storePresence 用 DataHub 里的 Agent 状态判断集群是否在线（指令服务据此拒绝发往无人接收的集群）
type storePresence struct {
	store datahub.Store
}

func (p storePresence) IsClusterOnline(clusterID string) bool {
	status, err := p.store.GetAgentStatus(clusterID)
	return err == nil && status != nil && status.IsOnline()
}

// runCommandHistoryRetention 每小时删除一次超过保留期的指令历史，ctx 结束即退出
func (m *Master) runCommandHistoryRetention(ctx context.Context) {
	if m.commandRetention <= 0 {
		log.Warn("指令历史保留期未配置，不做清理")
		return
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		deleted, err := m.commandOps.PurgeHistory(ctx, time.Now(), m.commandRetention)
		if err != nil {
			log.Warn("清理指令历史失败", "err", err)
		} else if deleted > 0 {
			log.Info("已清理过期指令历史", "rows", deleted, "retention", m.commandRetention.String())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
