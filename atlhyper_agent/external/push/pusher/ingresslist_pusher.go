// internal/push/pusher/ingresslist_pusher.go
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
	modelingress "AtlHyper/model/ingress"
)

// 区分数据来源类型（Envelope.Source）
// 请确保在 model/Source.go 中声明：const SourceIngressListSnapshot = "ingress_list_snapshot"
const SourceIngressListSnapshot = model.SourceIngressListSnapshot

// 载荷结构（与 Pod/Node/Service/Namespace/Metrics 保持一致）
type IngressListPayload struct {
	Ingresses []modelingress.Ingress `json:"ingresses"`
}

// PushIngressList 拉取全集群 Ingress 模型并上报；返回上报条数（Ingress 数）
func PushIngressList(ctx context.Context, clusterID, path string) (int, error) {
	list, err := cluster.IngressList(ctx)
	if err != nil {
		return 0, fmt.Errorf("cluster.IngressList: %w", err)
	}
	if len(list) == 0 {
		// 与其他 pusher 保持一致：无数据时静默跳过
		return 0, nil
	}

	payload := IngressListPayload{Ingresses: list}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal ingresslist payload: %w", err)
	}

	env := pushutils.NewEnvelope(clusterID, SourceIngressListSnapshot, raw)

	cfg := restcfg.NewDefaultRestClientConfig()
	cfg.Path = path
	sender := client.NewSender(cfg)

	code, resp, postErr := sender.Post(ctx, env)
	if postErr == nil && code >= 200 && code < 300 {
		// 成功：不打日志（与其他 pusher 一致）
		return len(list), nil
	}

	// 失败：向上传递错误（由调用方统一记录/告警）
	return 0, fmt.Errorf("post ingresslist: http=%d err=%v resp=%s", code, postErr, string(resp))
}

// StartIngressListPusher 周期性上报（interval<=0 时默认 30s；可按需调整）
func StartIngressListPusher(clusterID, path string, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	// 先推一轮，避免等待第一个 tick
	if _, err := PushIngressList(context.Background(), clusterID, path); err != nil {
		log.Printf("[ingresslist_pusher] error: %v", err)
	}

	go func() {
		t := time.NewTicker(interval)
		// 不 Stop；与其他 pusher 保持常驻风格
		for range t.C {
			if _, err := PushIngressList(context.Background(), clusterID, path); err != nil {
				log.Printf("[ingresslist_pusher] error: %v", err)
			}
		}
	}()
}
