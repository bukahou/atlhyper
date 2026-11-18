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
	"AtlHyper/model_v3/cluster"
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

	// 上次成功写库时各事件的状态（clusterID → 去重键 → 指纹）。
	// 快照每 10 秒到一份而事件多数没变，据此只写有变化的事件。
	persistedMu sync.Mutex
	persisted   map[string]map[string]eventFingerprint

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// eventFingerprint 判断一条事件相对上次写库是否有变化
type eventFingerprint struct {
	count        int32
	lastUnixNano int64 // 用时间戳数值比较：每份快照解码出的 time.Time 时区指针不同，直接比较会恒不相等
	message      string
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
		persisted:       make(map[string]map[string]eventFingerprint),
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

	// 2. 过滤 Warning 事件，按去重键合并
	current := mergeWarningEvents(clusterID, events)

	// 3. 只留下相对上次写库有变化的事件
	s.persistedMu.Lock()
	defer s.persistedMu.Unlock()

	previous := s.persisted[clusterID]
	changed := make([]*database.ClusterEvent, 0)
	for dedupKey, event := range current {
		if previous[dedupKey] != fingerprintOf(event) {
			changed = append(changed, event)
		}
	}

	// 4. 批量 UPSERT
	if len(changed) > 0 {
		if err := s.eventRepo.UpsertBatch(ctx, changed); err != nil {
			log.Error("Warning 事件写入失败",
				"cluster", clusterID,
				"count", len(changed),
				"err", err,
			)
			return err // 不更新指纹，下一份快照重试
		}
		// 正常同步使用 DEBUG 级别（避免刷屏）
		log.Debug("Warning 事件同步完成",
			"cluster", clusterID,
			"count", len(changed),
		)
	}

	// 5. 记录本次状态；快照里已经没有的事件随之移出，表的大小以当前快照为界
	fingerprints := make(map[string]eventFingerprint, len(current))
	for dedupKey, event := range current {
		fingerprints[dedupKey] = fingerprintOf(event)
	}
	s.persisted[clusterID] = fingerprints
	return nil
}

// mergeWarningEvents 取出 Warning 事件并按去重键合并。
//
// 同一对象同一原因在 K8s 里可能对应多条事件（消息不同，各自计数）：
// 次数相加，首次时间取最早，其余字段取最后发生的那一条。
func mergeWarningEvents(clusterID string, events []cluster.Event) map[string]*database.ClusterEvent {
	merged := make(map[string]*database.ClusterEvent)
	for i := range events {
		e := &events[i]
		if e.Type != "Warning" {
			continue
		}

		// 去重键: MD5(cluster_id + involved_kind + involved_namespace + involved_name + reason)
		dedupKey := generateDedupKey(clusterID, e.InvolvedObject.Kind, e.InvolvedObject.Namespace, e.InvolvedObject.Name, e.Reason)

		existing, ok := merged[dedupKey]
		if !ok {
			merged[dedupKey] = &database.ClusterEvent{
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
			}
			continue
		}

		existing.Count += e.Count
		if e.FirstTimestamp.Before(existing.FirstTimestamp) {
			existing.FirstTimestamp = e.FirstTimestamp
		}
		if e.LastTimestamp.After(existing.LastTimestamp) {
			existing.LastTimestamp = e.LastTimestamp
			existing.Name = e.Name
			existing.Namespace = e.Namespace
			existing.Message = e.Message
			existing.SourceComponent = e.Source
		}
	}
	return merged
}

func fingerprintOf(event *database.ClusterEvent) eventFingerprint {
	return eventFingerprint{
		count:        event.Count,
		lastUnixNano: event.LastTimestamp.UnixNano(),
		message:      event.Message,
	}
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
