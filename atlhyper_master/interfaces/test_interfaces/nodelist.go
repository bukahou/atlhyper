// external/interfaces/nodelist.go
package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model"
	modelnode "AtlHyper/model/node"
)

const sourceNodeList = model.SourceNodeListSnapshot

// 兼容两种载荷：{"nodes":[...]} 或直接数组 [...]
type nodeListPayload struct {
	Nodes []modelnode.Node `json:"nodes"`
}

func decodeNodeListPayload(raw []byte) ([]modelnode.Node, error) {
	// 1) 对象形式
	var obj nodeListPayload
	if err := json.Unmarshal(raw, &obj); err == nil {
		// 允许空数组；用于“完整但为空”的情况
		return obj.Nodes, nil
	}
	// 2) 直接数组
	var arr []modelnode.Node
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("payload 中未找到 nodes")
}

// GetLatestNodeListByCluster
// 说明：由于 EnvelopeRecord 未暴露时间戳，这里取“最后一条匹配记录”的解码结果作为最新。
// 若后续 master_store 提供 ts，可改为按 ts 选最大者。
func GetLatestNodeListByCluster(_ context.Context, clusterID string) ([]modelnode.Node, error) {
	if clusterID == "" {
		return []modelnode.Node{}, nil
	}
	var latest []modelnode.Node

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceNodeList || r.ClusterID != clusterID {
			continue
		}
		nodes, err := decodeNodeListPayload(r.Payload)
		if err != nil {
			log.Printf("[node_iface] decode payload fail: cluster=%s err=%v", clusterID, err)
			continue
		}
		latest = nodes // 保留最后一条（到达顺序）
	}
	if latest == nil {
		return []modelnode.Node{}, nil
	}
	return latest, nil
}
