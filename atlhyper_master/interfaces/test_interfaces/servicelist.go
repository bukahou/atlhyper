// NeuroController/external/interfaces/servicelist.go
package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model"
	modelservice "AtlHyper/model/service"
)

const sourceServiceList = model.SourceServiceListSnapshot

// 兼容两种载荷：{"services":[...]} 或直接数组 [...]
type serviceListPayload struct {
	Services []modelservice.Service `json:"services"`
}

func decodeServiceListPayload(raw []byte) ([]modelservice.Service, error) {
	// 1) 对象形式
	var obj serviceListPayload
	if err := json.Unmarshal(raw, &obj); err == nil {
		// 允许空数组；用于“完整但为空”的情况
		return obj.Services, nil
	}
	// 2) 直接数组
	var arr []modelservice.Service
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("payload 中未找到 services")
}

// GetLatestServiceListByCluster
// 说明：由于 EnvelopeRecord 未暴露时间戳，这里取“最后一条匹配记录”的解码结果作为最新。
// 若后续 master_store 提供 ts，可改为按 ts 选最大者。
func GetLatestServiceListByCluster(_ context.Context, clusterID string) ([]modelservice.Service, error) {
	if clusterID == "" {
		return []modelservice.Service{}, nil
	}
	var latest []modelservice.Service

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceServiceList || r.ClusterID != clusterID {
			continue
		}
		svcs, err := decodeServiceListPayload(r.Payload)
		if err != nil {
			log.Printf("[service_iface] decode payload fail: cluster=%s err=%v", clusterID, err)
			continue
		}
		latest = svcs // 保留最后一条（到达顺序）
	}
	if latest == nil {
		return []modelservice.Service{}, nil
	}
	return latest, nil
}