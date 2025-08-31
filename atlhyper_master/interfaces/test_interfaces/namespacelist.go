// NeuroController/external/interfaces/namespacelist.go
package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model"
	modelns "AtlHyper/model/namespace"
)

const sourceNamespaceList = model.SourceNamespaceListSnapshot

// 兼容两种载荷：{"namespaces":[...]} 或直接数组 [...]
type namespaceListPayload struct {
	Namespaces []modelns.Namespace `json:"namespaces"`
}

func decodeNamespaceListPayload(raw []byte) ([]modelns.Namespace, error) {
	// 1) 对象形式
	var obj namespaceListPayload
	if err := json.Unmarshal(raw, &obj); err == nil {
		// 允许空数组；用于“完整但为空”的情况
		return obj.Namespaces, nil
	}
	// 2) 直接数组
	var arr []modelns.Namespace
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("payload 中未找到 namespaces")
}

// GetLatestNamespaceListByCluster
// 说明：由于 EnvelopeRecord 未暴露时间戳，这里按“到达顺序最后一条”为最新。
// 若后续 master_store 提供 ts，可改为按时间最大者。
func GetLatestNamespaceListByCluster(_ context.Context, clusterID string) ([]modelns.Namespace, error) {
	if clusterID == "" {
		return []modelns.Namespace{}, nil
	}
	var latest []modelns.Namespace

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceNamespaceList || r.ClusterID != clusterID {
			continue
		}
		nss, err := decodeNamespaceListPayload(r.Payload)
		if err != nil {
			log.Printf("[namespace_iface] decode payload fail: cluster=%s err=%v", clusterID, err)
			continue
		}
		latest = nss // 保留最后一条（到达顺序）
	}
	if latest == nil {
		return []modelns.Namespace{}, nil
	}
	return latest, nil
}
