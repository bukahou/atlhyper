// internal/push/pusher/deployment/deploymentlist_pusher.go
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
	"AtlHyper/model/transport"
	modeldeployment "AtlHyper/model/k8s"
)

// 区分数据来源类型（Envelope.Source）
// 请在 model/Source.go 中声明：const SourceDeploymentListSnapshot = "deployment_list_snapshot"
const SourceDeploymentListSnapshot = transport.SourceDeploymentListSnapshot

// 载荷结构（与 Pod/Node/Service/Metrics 保持一致）
type DeploymentListPayload struct {
	Deployments []modeldeployment.Deployment `json:"deployments"`
}

// PushDeploymentList 拉取全集群 Deployment 模型并上报；返回上报条数（Deployment 数）
func PushDeploymentList(ctx context.Context, clusterID, path string) (int, error) {
	deploys, err := cluster.DeploymentList(ctx)
	if err != nil {
		return 0, fmt.Errorf("cluster.DeploymentList: %w", err)
	}
	if len(deploys) == 0 {
		// 与其他 pusher 保持一致：无数据时静默跳过
		return 0, nil
	}

	payload := DeploymentListPayload{Deployments: deploys}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal deploymentlist payload: %w", err)
	}

	env := pushutils.NewEnvelope(clusterID, SourceDeploymentListSnapshot, raw)

	cfg := restcfg.NewDefaultRestClientConfig()
	cfg.Path = path
	sender := client.NewSender(cfg)

	code, resp, postErr := sender.Post(ctx, env)
	if postErr == nil && code >= 200 && code < 300 {
		// 成功：不打日志，保持与其他 pusher 一致
		return len(deploys), nil
	}

	// 失败：向上传递错误（由调用方统一记录/告警）
	return 0, fmt.Errorf("post deploymentlist: http=%d err=%v resp=%s", code, postErr, string(resp))
}

// StartDeploymentListPusher 周期性上报（interval<=0 时默认 30s；可按需调整）
func StartDeploymentListPusher(clusterID, path string, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	// 先推一轮，避免等待第一个 tick
	if _, err := PushDeploymentList(context.Background(), clusterID, path); err != nil {
		log.Printf("[deploymentlist_pusher] error: %v", err)
	}

	go func() {
		t := time.NewTicker(interval)
		// 不 Stop；与其他 pusher 保持常驻风格
		for range t.C {
			if _, err := PushDeploymentList(context.Background(), clusterID, path); err != nil {
				log.Printf("[deploymentlist_pusher] error: %v", err)
			}
		}
	}()
}
