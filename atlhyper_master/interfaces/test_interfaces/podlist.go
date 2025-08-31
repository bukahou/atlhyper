// NeuroController/external/interfaces/podlist.go
package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model"
	modelpod "AtlHyper/model/pod"
)

const sourcePodList = model.SourcePodListSnapshot

// 兼容两种载荷：{"pods":[...]} 或直接数组 [...]
type podListPayload struct {
	Pods []modelpod.Pod `json:"pods"`
}

func decodePodListPayload(raw []byte) ([]modelpod.Pod, error) {
	// 1) 对象形式
	var obj podListPayload
	if err := json.Unmarshal(raw, &obj); err == nil {
		// 允许空数组；用于“完整但为空”的情况
		return obj.Pods, nil
	}
	// 2) 直接数组
	var arr []modelpod.Pod
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("payload 中未找到 pods")
}

// GetLatestPodListByCluster
// 说明：由于 EnvelopeRecord 未暴露时间戳，这里取“最后一条匹配记录”的解码结果作为最新。
// 若后续 master_store 提供 ts，可改为按 ts 选最大者。
func GetLatestPodListByCluster(_ context.Context, clusterID string) ([]modelpod.Pod, error) {
	if clusterID == "" {
		return []modelpod.Pod{}, nil
	}
	var latest []modelpod.Pod

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourcePodList || r.ClusterID != clusterID {
			continue
		}
		pods, err := decodePodListPayload(r.Payload)
		if err != nil {
			log.Printf("[pod_iface] decode payload fail: cluster=%s err=%v", clusterID, err)
			continue
		}
		latest = pods // 保留最后一条（到达顺序）
	}
	if latest == nil {
		return []modelpod.Pod{}, nil
	}
	return latest, nil
}
