// internal/push/pusher/nodelist_pusher.go
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
	modelnode "AtlHyper/model/node"
)

// 区分数据来源类型（Envelope.Source）
// 需要在 model/Source.go 中声明：const SourceNodeListSnapshot = "node_list_snapshot"
const SourceNodeListSnapshot = model.SourceNodeListSnapshot

// 载荷结构（与 Pod/Metrics 保持一致）
type NodeListPayload struct {
	Nodes []modelnode.Node `json:"nodes"`
}

// PushNodeList 拉取全集群 Node 模型并上报；返回上报条数（Node 数）
func PushNodeList(ctx context.Context, clusterID, path string) (int, error) {
	nodes, err := cluster.NodeList(ctx)
	if err != nil {
		return 0, fmt.Errorf("cluster.NodeList: %w", err)
	}
	if len(nodes) == 0 {
		// 与 metrics/podlist 一致：无数据时静默跳过
		return 0, nil
	}

	payload := NodeListPayload{Nodes: nodes}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal nodelist payload: %w", err)
	}

	env := pushutils.NewEnvelope(clusterID, SourceNodeListSnapshot, raw)

	cfg := restcfg.NewDefaultRestClientConfig()
	cfg.Path = path
	sender := client.NewSender(cfg)

	code, resp, postErr := sender.Post(ctx, env)
	if postErr == nil && code >= 200 && code < 300 {
		// 成功：不打日志，保持与 metrics/podlist 一致
		return len(nodes), nil
	}

	// 失败：向上传递错误（由调用方统一记录/告警）
	return 0, fmt.Errorf("post nodelist: http=%d err=%v resp=%s", code, postErr, string(resp))
}

// StartNodeListPusher 周期性上报（interval<=0 时默认 30s；可按需调整）
func StartNodeListPusher(clusterID, path string, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	// 先推一轮，避免等待第一个 tick
	if _, err := PushNodeList(context.Background(), clusterID, path); err != nil {
		log.Printf("[nodelist_pusher] error: %v", err)
	}

	go func() {
		t := time.NewTicker(interval)
		// 不 Stop；与其他 pusher 保持常驻风格
		for range t.C {
			if _, err := PushNodeList(context.Background(), clusterID, path); err != nil {
				log.Printf("[nodelist_pusher] error: %v", err)
			}
		}
	}()
}
