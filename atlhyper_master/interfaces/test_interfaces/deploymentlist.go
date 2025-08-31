// NeuroController/external/interfaces/deploymentlist.go
package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model"
	modeldeploy "AtlHyper/model/deployment"
)

const sourceDeploymentList = model.SourceDeploymentListSnapshot

// 兼容两种载荷：{"deployments":[...]} 或直接数组 [...]
type deploymentListPayload struct {
	Deployments []modeldeploy.Deployment `json:"deployments"`
}

func decodeDeploymentListPayload(raw []byte) ([]modeldeploy.Deployment, error) {
	// 1) 对象形式
	var obj deploymentListPayload
	if err := json.Unmarshal(raw, &obj); err == nil {
		// 允许空数组；用于“完整但为空”的情况
		return obj.Deployments, nil
	}
	// 2) 直接数组
	var arr []modeldeploy.Deployment
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("payload 中未找到 deployments")
}

// GetLatestDeploymentListByCluster
// 说明：由于 EnvelopeRecord 未暴露时间戳，这里取“最后一条匹配记录”的解码结果作为最新。
// 若后续 master_store 提供 ts，可改为按 ts 选最大者。
func GetLatestDeploymentListByCluster(_ context.Context, clusterID string) ([]modeldeploy.Deployment, error) {
	if clusterID == "" {
		return []modeldeploy.Deployment{}, nil
	}
	var latest []modeldeploy.Deployment

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceDeploymentList || r.ClusterID != clusterID {
			continue
		}
		deps, err := decodeDeploymentListPayload(r.Payload)
		if err != nil {
			log.Printf("[deployment_iface] decode payload fail: cluster=%s err=%v", clusterID, err)
			continue
		}
		latest = deps // 保留最后一条（到达顺序）
	}
	if latest == nil {
		return []modeldeploy.Deployment{}, nil
	}
	return latest, nil
}
