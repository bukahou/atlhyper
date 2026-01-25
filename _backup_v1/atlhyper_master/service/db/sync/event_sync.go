// service/sync/event_sync.go
package sync

import (
	"context"
	"log"
	"time"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/model/transport"
)

// EventSyncService 事件同步服务
// -----------------------------------------------------------------------------
// 职责：
//   - 从 DataHub 读取事件数据
//   - 去重处理
//   - 写入 SQLite 持久化
// 数据流：DataHub → Repository → Service → PersistenceRepository → SQLite
// -----------------------------------------------------------------------------
type EventSyncService struct {
	deduper *EventDeduplicator
}

// NewEventSyncService 创建事件同步服务实例
func NewEventSyncService() *EventSyncService {
	return &EventSyncService{
		deduper: NewEventDeduplicator(),
	}
}

// 全局单例
var defaultEventSyncService *EventSyncService

// InitEventSync 初始化全局事件同步服务
func InitEventSync() {
	defaultEventSyncService = NewEventSyncService()
}

// DefaultEventSync 获取全局事件同步服务实例
func DefaultEventSync() *EventSyncService {
	if defaultEventSyncService == nil {
		defaultEventSyncService = NewEventSyncService()
	}
	return defaultEventSyncService
}

// SyncEventsToDatabase 同步事件到数据库
// -----------------------------------------------------------------------------
// 流程：
//  1. 通过 Repository 读取 DataHub 中的事件
//  2. 使用 Deduplicator 去重
//  3. 通过 Persistence Repository 写入 SQLite
// -----------------------------------------------------------------------------
func (s *EventSyncService) SyncEventsToDatabase(ctx context.Context) error {
	// 1. 获取所有集群 ID
	clusterIDs, err := repository.Mem.ListClusterIDs(ctx)
	if err != nil {
		return err
	}

	if len(clusterIDs) == 0 {
		// 无集群数据，清空去重缓存
		s.deduper.Clear()
		return nil
	}

	// 2. 遍历每个集群，读取事件
	allEvents := make([]transport.LogEvent, 0)
	clusterEventMap := make(map[string][]transport.LogEvent)

	for _, clusterID := range clusterIDs {
		events, err := repository.Mem.GetK8sEventsRecent(ctx, clusterID, 1000)
		if err != nil {
			log.Printf("[sync] 读取集群 %s 事件失败: %v", clusterID, err)
			continue
		}
		if len(events) > 0 {
			clusterEventMap[clusterID] = events
			allEvents = append(allEvents, events...)
		}
	}

	if len(allEvents) == 0 {
		s.deduper.Clear()
		return nil
	}

	// 3. 去重处理
	newEvents := s.deduper.Filter(clusterEventMap)
	if len(newEvents) == 0 {
		return nil
	}

	// 4. 写入 SQLite（通过 Persistence Repository）
	for _, ev := range newEvents {
		eventLogRow := transport.EventLog{
			ClusterID: ev.ClusterID,
			Category:  ev.Category,
			EventTime: ev.Timestamp.Format(time.RFC3339),
			Kind:      ev.Kind,
			Message:   ev.Message,
			Name:      ev.Name,
			Namespace: ev.Namespace,
			Node:      ev.Node,
			Reason:    ev.ReasonCode,
			Severity:  ev.Severity,
			Time:      time.Now().Format(time.RFC3339),
		}
		if err := repository.Event.Insert(ctx, &eventLogRow); err != nil {
			log.Printf("[sync] 插入事件失败: %v", err)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// EventDeduplicator 事件去重器
// -----------------------------------------------------------------------------
type EventDeduplicator struct {
	cache map[string]eventCacheRecord
}

type eventCacheRecord struct {
	Message  string
	Severity string
	Category string
}

// NewEventDeduplicator 创建去重器
func NewEventDeduplicator() *EventDeduplicator {
	return &EventDeduplicator{
		cache: make(map[string]eventCacheRecord),
	}
}

// Clear 清空缓存
func (d *EventDeduplicator) Clear() {
	d.cache = make(map[string]eventCacheRecord)
}

// Filter 过滤重复事件，返回新事件列表
// -----------------------------------------------------------------------------
// 去重逻辑：
//   - key = ClusterID + Kind + Namespace + Name + Reason + Message
//   - 如果 Message/Severity/Category 有变化，则认为是新事件
// -----------------------------------------------------------------------------
func (d *EventDeduplicator) Filter(clusterEventMap map[string][]transport.LogEvent) []eventWithCluster {
	newEvents := make([]eventWithCluster, 0)

	for clusterID, events := range clusterEventMap {
		for _, ev := range events {
			cacheKey := clusterID + "|" + ev.Kind + "|" + ev.Namespace + "|" +
				ev.Name + "|" + ev.ReasonCode + "|" + ev.Message

			last, exists := d.cache[cacheKey]
			changed := !exists ||
				ev.Message != last.Message ||
				ev.Severity != last.Severity ||
				ev.Category != last.Category

			if !changed {
				continue
			}

			d.cache[cacheKey] = eventCacheRecord{
				Message:  ev.Message,
				Severity: ev.Severity,
				Category: ev.Category,
			}

			newEvents = append(newEvents, eventWithCluster{
				ClusterID: clusterID,
				LogEvent:  ev,
			})
		}
	}

	return newEvents
}

// eventWithCluster 带集群ID的事件
type eventWithCluster struct {
	ClusterID string
	transport.LogEvent
}
