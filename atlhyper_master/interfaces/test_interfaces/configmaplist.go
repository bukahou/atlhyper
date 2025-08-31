// NeuroController/external/interfaces/configmaplist.go
package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model"
	modelcm "AtlHyper/model/configmap"
)

const sourceConfigMapList = model.SourceConfigMapListSnapshot

// 兼容两种载荷：{"configmaps":[...]} 或直接数组 [...]
type configMapListPayload struct {
	ConfigMaps []modelcm.ConfigMap `json:"configmaps"`
}

func decodeConfigMapListPayload(raw []byte) ([]modelcm.ConfigMap, error) {
	// 1) 对象形式
	var obj configMapListPayload
	if err := json.Unmarshal(raw, &obj); err == nil {
		// 允许空数组；用于“完整但为空”的情况
		return obj.ConfigMaps, nil
	}
	// 2) 直接数组
	var arr []modelcm.ConfigMap
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("payload 中未找到 configmaps")
}

// GetLatestConfigMapListByCluster
// 说明：当前 master_store 通过 ReplaceLatest 仅保留同 cluster+source 的最新一帧，
// 这里依然遍历以保持与其他接口一致的写法。如后续改为时间筛选，此处无需变更签名。
func GetLatestConfigMapListByCluster(_ context.Context, clusterID string) ([]modelcm.ConfigMap, error) {
	if clusterID == "" {
		return []modelcm.ConfigMap{}, nil
	}
	var latest []modelcm.ConfigMap

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceConfigMapList || r.ClusterID != clusterID {
			continue
		}
		cms, err := decodeConfigMapListPayload(r.Payload)
		if err != nil {
			log.Printf("[configmap_iface] decode payload fail: cluster=%s err=%v", clusterID, err)
			continue
		}
		latest = cms // 保留最后一条（按到达顺序）
	}
	if latest == nil {
		return []modelcm.ConfigMap{}, nil
	}
	return latest, nil
}
