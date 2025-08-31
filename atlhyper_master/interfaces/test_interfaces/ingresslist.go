package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model"
	modelingr "AtlHyper/model/ingress"
)

const sourceIngressList = model.SourceIngressListSnapshot

// 兼容两种载荷：{"ingresses":[...]} 或直接数组 [...]
type ingressListPayload struct {
	Ingresses []modelingr.Ingress `json:"ingresses"`
}

func decodeIngressListPayload(raw []byte) ([]modelingr.Ingress, error) {
	var obj ingressListPayload
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Ingresses, nil
	}
	var arr []modelingr.Ingress
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("payload 中未找到 ingresses")
}

// GetLatestIngressListByCluster —— 只返回该 cluster 的“最新一帧” Ingress 列表
func GetLatestIngressListByCluster(_ context.Context, clusterID string) ([]modelingr.Ingress, error) {
	if clusterID == "" {
		return []modelingr.Ingress{}, nil
	}
	var latest []modelingr.Ingress

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceIngressList || r.ClusterID != clusterID {
			continue
		}
		ings, err := decodeIngressListPayload(r.Payload)
		if err != nil {
			log.Printf("[ingress_iface] decode payload fail: cluster=%s err=%v", clusterID, err)
			continue
		}
		latest = ings
	}
	if latest == nil {
		return []modelingr.Ingress{}, nil
	}
	return latest, nil
}
