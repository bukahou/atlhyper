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

// retentionJob 一类只增不减的数据：保留多久、怎么删
type retentionJob struct {
	name  string
	keep  time.Duration // <= 0 表示不清理
	purge func(ctx context.Context, before time.Time) (int64, error)
}

// runRetention 启动时清理一次，之后每小时一次；ctx 结束即退出
func runRetention(ctx context.Context, jobs []retentionJob) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		purgeOnce(ctx, jobs, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// purgeOnce 每类按自己的保留期删一次；一类出错只记日志，不影响其他类
func purgeOnce(ctx context.Context, jobs []retentionJob, now time.Time) {
	for _, job := range jobs {
		if job.keep <= 0 {
			continue
		}
		deleted, err := job.purge(ctx, now.Add(-job.keep))
		switch {
		case err != nil:
			log.Warn("清理过期数据失败", "kind", job.name, "err", err)
		case deleted > 0:
			log.Info("已清理过期数据", "kind", job.name, "rows", deleted, "retention", job.keep.String())
		}
	}
}
