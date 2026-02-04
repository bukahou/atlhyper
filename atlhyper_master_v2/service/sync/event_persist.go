// atlhyper_master_v2/service/sync/event_persist.go
// Event 持久化服务
// 核心功能：将 DataHub 中的 Warning Events UPSERT 到 RDB（去重）
package sync

import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/common/logger"
)

var log = logger.Module("EventPersist")

// EventPersistService Event 持久化服务
type EventPersistService struct {
	store     datahub.Store
	eventRepo database.ClusterEventRepository

	// 配置
	retentionDays   int
	maxCount        int
	cleanupInterval time.Duration

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// EventPersistConfig 配置
type EventPersistConfig struct {
	RetentionDays   int
	MaxCount        int
	CleanupInterval time.Duration
}

// NewEventPersistService 创建服务
func NewEventPersistService(
	store datahub.Store,
	eventRepo database.ClusterEventRepository,
	cfg EventPersistConfig,
) *EventPersistService {
	return &EventPersistService{
		store:           store,
		eventRepo:       eventRepo,
		retentionDays:   cfg.RetentionDays,
		maxCount:        cfg.MaxCount,
		cleanupInterval: cfg.CleanupInterval,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动服务
func (s *EventPersistService) Start() error {
	// 启动清理协程
	s.wg.Add(1)
	go s.cleanupLoop()

	log.Info("事件持久化服务已启动")
	return nil
}

// Stop 停止服务
func (s *EventPersistService) Stop() error {
	close(s.stopCh)
	s.wg.Wait()
	log.Info("事件持久化服务已停止")
	return nil
}

// Sync 同步指定集群的 Warning Events 到 RDB
// 只持久化 Warning 类型事件，基于业务键去重
// 由快照到达时触发调用
func (s *EventPersistService) Sync(clusterID string) error {
	ctx := context.Background()

	// 1. 从 DataHub 获取当前集群所有 Events
	events, err := s.store.GetEvents(clusterID)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	// 2. 过滤 Warning 事件，转换为 Repository 格式
	repoEvents := make([]*database.ClusterEvent, 0)
	for _, e := range events {
		// 只持久化 Warning 事件
		if e.Type != "Warning" {
			continue
		}

		// 生成去重键: MD5(cluster_id + involved_kind + involved_namespace + involved_name + reason)
		dedupKey := generateDedupKey(clusterID, e.InvolvedObject.Kind, e.InvolvedObject.Namespace, e.InvolvedObject.Name, e.Reason)

		repoEvents = append(repoEvents, &database.ClusterEvent{
			DedupKey:          dedupKey,
			ClusterID:         clusterID,
			Namespace:         e.Namespace,
			Name:              e.Name,
			Type:              e.Type,
			Reason:            e.Reason,
			Message:           e.Message,
			SourceComponent:   e.Source,
			SourceHost:        "",
			InvolvedKind:      e.InvolvedObject.Kind,
			InvolvedName:      e.InvolvedObject.Name,
			InvolvedNamespace: e.InvolvedObject.Namespace,
			FirstTimestamp:    e.FirstTimestamp,
			LastTimestamp:     e.LastTimestamp,
			Count:             e.Count,
		})
	}

	if len(repoEvents) == 0 {
		return nil
	}

	// 3. 批量 UPSERT
	if err := s.eventRepo.UpsertBatch(ctx, repoEvents); err != nil {
		log.Error("Warning 事件写入失败",
			"cluster", clusterID,
			"count", len(repoEvents),
			"err", err,
		)
		return err
	}

	// 正常同步使用 DEBUG 级别（避免刷屏）
	log.Debug("Warning 事件同步完成",
		"cluster", clusterID,
		"count", len(repoEvents),
	)
	return nil
}

// generateDedupKey 生成去重键
// 格式: MD5(cluster_id + involved_kind + involved_namespace + involved_name + reason)
func generateDedupKey(clusterID, kind, namespace, name, reason string) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%s", clusterID, kind, namespace, name, reason)
	hash := md5.Sum([]byte(raw))
	return fmt.Sprintf("%x", hash)
}

// cleanupLoop 定期清理过期事件
func (s *EventPersistService) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup 执行清理
func (s *EventPersistService) cleanup() {
	ctx := context.Background()

	// 获取所有 Agent
	agents, err := s.store.ListAgents()
	if err != nil {
		log.Error("获取 Agent 列表失败", "err", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -s.retentionDays)

	for _, agent := range agents {
		clusterID := agent.ClusterID

		// 1. 删除过期事件
		deleted, err := s.eventRepo.DeleteBefore(ctx, clusterID, cutoff)
		if err != nil {
			log.Error("删除过期事件失败", "cluster", clusterID, "err", err)
			continue
		}
		if deleted > 0 {
			log.Info("已删除过期事件", "cluster", clusterID, "count", deleted)
		}

		// 2. 检查是否超过最大数量
		count, err := s.eventRepo.CountByCluster(ctx, clusterID)
		if err != nil {
			continue
		}

		if count > int64(s.maxCount) {
			deleted, err := s.eventRepo.DeleteOldest(ctx, clusterID, s.maxCount)
			if err != nil {
				log.Error("删除最旧事件失败", "cluster", clusterID, "err", err)
				continue
			}
			if deleted > 0 {
				log.Info("已删除最旧事件", "cluster", clusterID, "count", deleted)
			}
		}
	}
}
