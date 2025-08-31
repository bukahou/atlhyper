// internal/push/pusher/metrics_pusher.go
package pusher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"AtlHyper/atlhyper_agent/external/push/client"
	restcfg "AtlHyper/atlhyper_agent/external/push/config"
	pushutils "AtlHyper/atlhyper_agent/external/push/utils"
	dataapi "AtlHyper/atlhyper_agent/interfaces/data_api"
	"AtlHyper/model"
	"AtlHyper/model/metrics"
)

const SourceMetricsSnapshot = model.SourceMetricsSnapshot

type MetricsPayload struct {
	Snapshots []metrics.NodeMetricsSnapshot `json:"snapshots"`
}

func PushLatestMetrics(ctx context.Context, clusterID, path string) (int, error) {
	latest := dataapi.GetAllLatestNodeMetrics()
	if len(latest) == 0 {
		// 静默跳过：不打日志
		return 0, nil
	}

	payload := MetricsPayload{
		Snapshots: make([]metrics.NodeMetricsSnapshot, 0, len(latest)),
	}
	for _, snap := range latest {
		payload.Snapshots = append(payload.Snapshots, snap)
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}
	env := pushutils.NewEnvelope(clusterID, SourceMetricsSnapshot, raw)

	cfg := restcfg.NewDefaultRestClientConfig()
	cfg.Path = path
	sender := client.NewSender(cfg)

	code, resp, postErr := sender.Post(ctx, env)
	if postErr == nil && code >= 200 && code < 300 {
		// 成功：不打日志
		return len(payload.Snapshots), nil
	}

	// 失败：向上传递错误（统一由调用方记录）
	return 0, fmt.Errorf("post metrics: http=%d err=%v resp=%s", code, postErr, string(resp))
}

// StartMetricsPusher 周期性上报（interval<=0 时默认 10s）
func StartMetricsPusher(clusterID, path string, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	// 先推一轮，避免等第一个 tick
	if _, err := PushLatestMetrics(context.Background(), clusterID, path); err != nil {
		log.Printf("[metrics_pusher] error: %v", err)
	}

	go func() {
		t := time.NewTicker(interval)
		// 不调用 t.Stop()；协程常驻到进程退出
		for range t.C {
			if _, err := PushLatestMetrics(context.Background(), clusterID, path); err != nil {
				log.Printf("[metrics_pusher] error: %v", err)
			}
		}
	}()
}