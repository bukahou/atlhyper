// internal/push/pusher/podlist_pusher.go
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
	"AtlHyper/atlhyper_agent/interfaces/cluster"
	"AtlHyper/model"
	modelpod "AtlHyper/model/pod"
)

// 区分数据来源类型（Envelope.Source）
const SourcePodListSnapshot = model.SourcePodListSnapshot

// 载荷结构（和 metrics_pusher 风格一致，简单明了）
type PodListPayload struct {
	Pods []modelpod.Pod `json:"pods"`
}

// PushPodList 拉取全集群 Pod 模型并上报；返回上报条数（Pod 数）
func PushPodList(ctx context.Context, clusterID, path string) (int, error) {
	pods, err := cluster.PodList(ctx)
	if err != nil {
		return 0, fmt.Errorf("cluster.PodList: %w", err)
	}
	if len(pods) == 0 {
		// 和 metrics_pusher 一致：无数据时静默跳过
		return 0, nil
	}

	payload := PodListPayload{Pods: pods}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal podlist payload: %w", err)
	}

	env := pushutils.NewEnvelope(clusterID, SourcePodListSnapshot, raw)

	cfg := restcfg.NewDefaultRestClientConfig()
	cfg.Path = path
	sender := client.NewSender(cfg)

	code, resp, postErr := sender.Post(ctx, env)
	if postErr == nil && code >= 200 && code < 300 {
		// 成功：不打日志，保持与 metrics_pusher 一致
		return len(pods), nil
	}

	// 失败：向上传递错误（由调用方统一记录/告警）
	return 0, fmt.Errorf("post podlist: http=%d err=%v resp=%s", code, postErr, string(resp))
}

// StartPodListPusher 周期性上报（interval<=0 时默认 30s；可按需调整）
func StartPodListPusher(clusterID, path string, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	// 先推一轮，避免等待第一个 tick
	if _, err := PushPodList(context.Background(), clusterID, path); err != nil {
		log.Printf("[podlist_pusher] error: %v", err)
	}

	go func() {
		t := time.NewTicker(interval)
		// 不 Stop；与 metrics_pusher 保持常驻风格
		for range t.C {
			if _, err := PushPodList(context.Background(), clusterID, path); err != nil {
				log.Printf("[podlist_pusher] error: %v", err)
			}
		}
	}()
}
