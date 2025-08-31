// internal/push/events_pusher.go
package pusher

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"AtlHyper/atlhyper_agent/external/push/client"
	pcfg "AtlHyper/atlhyper_agent/external/push/config"
	"AtlHyper/atlhyper_agent/external/push/utils"
	"AtlHyper/atlhyper_agent/interfaces"
	Source "AtlHyper/model"
	model "AtlHyper/model/event"
)

const SourceK8sEvent = Source.SourceK8sEvent

// lastSentMap 记录已见/已发事件的最新时间戳，单协程无需加锁
// key: Kind|Namespace|Name|ReasonCode|Message
// val: 最近一次见到该键的时间（用于增量判断）
var lastSentMap = make(map[string]time.Time)

// makeKey 生成“足够唯一”的去重键（含 Message，避免不同内容被混淆）
func makeKey(ev model.LogEvent) string {
	return ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode + "|" + ev.Message
}

// PushEvents：获取 → 增量过滤 → 打包 → 发送
// 需要显式传入 path（例如 pcfg.PathEventsCleaned）
func PushEvents(ctx context.Context, clusterID string, path string) (int, error) {
	events := interfaces.GetCleanedEventLogs()
	if len(events) == 0 {
		return 0, nil
	}

	newEvents := make([]model.LogEvent, 0, len(events))
	currentKeys := make(map[string]struct{}, len(events))
	for _, ev := range events {
		key := makeKey(ev)
		currentKeys[key] = struct{}{}
		if ts, ok := lastSentMap[key]; !ok || ev.Timestamp.After(ts) {
			newEvents = append(newEvents, ev)
			lastSentMap[key] = ev.Timestamp
		}
	}
	for k := range lastSentMap {
		if _, still := currentKeys[k]; !still {
			delete(lastSentMap, k)
		}
	}
	if len(newEvents) == 0 {
		return 0, nil
	}

	payload, err := json.Marshal(map[string]any{"events": newEvents})
	if err != nil {
		return 0, err
	}

	// 使用常量 SourceK8sEvent
	env := utils.NewEnvelope(clusterID, SourceK8sEvent, payload)

	restCfg := pcfg.NewDefaultRestClientConfig()
	restCfg.Path = path
	sender := client.NewSender(restCfg)

	code, _, postErr := sender.Post(ctx, env)
	if postErr == nil && code >= 200 && code < 300 {
		log.Printf("event_push ok size=%d code=%d", len(newEvents), code)
		return len(newEvents), nil
	}
	log.Printf("event_push fail code=%d err=%v", code, postErr)
	return 0, postErr
}

// StartEventsPusher：把循环放回 pusher 包（与 StartMetricsPusher 同风格）
func StartEventsPusher(clusterID, path string, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	// 先推一次，避免等第一个 tick
	if _, err := PushEvents(context.Background(), clusterID, path); err != nil {
		log.Printf("[events_pusher] error: %v", err)
	}

	go func() {
		t := time.NewTicker(interval)
		// 不需要 defer t.Stop()：协程常驻到进程结束
		for range t.C {
			if _, err := PushEvents(context.Background(), clusterID, path); err != nil {
				log.Printf("[events_pusher] error: %v", err)
			}
		}
	}()
}
