// atlhyper_master_v2/agentsdk/snapshot.go
// 处理 Agent 快照上报
// 直接解析 cluster.ClusterSnapshot 格式
package agentsdk

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"AtlHyper/common"
	"AtlHyper/model_v3/cluster"
)

// 使用 server.go 中定义的 log 变量

// handleSnapshot 处理快照上报
// POST /agent/snapshot
func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 压缩前后都限制大小：防止超大请求体或高压缩比的内容耗尽内存
	limit := s.snapshotLimit()
	r.Body = http.MaxBytesReader(w, r.Body, limit)

	// 解压 Gzip（如果有）
	reader, err := common.MaybeGunzipReaderAuto(r.Body, r.Header.Get("Content-Encoding"))
	if err != nil {
		log.Error("解压请求体失败", "err", err)
		http.Error(w, "Invalid gzip", http.StatusBadRequest)
		return
	}
	defer reader.Close()

	// 直接解析为 cluster.ClusterSnapshot
	var snapshot cluster.ClusterSnapshot
	if err := json.NewDecoder(&cappedReader{source: reader, remaining: limit}).Decode(&snapshot); err != nil {
		if isBodyTooLarge(err) {
			log.Warn("快照超过大小上限，已拒绝", "limitBytes", limit)
			http.Error(w, "Snapshot too large", http.StatusRequestEntityTooLarge)
			return
		}
		log.Error("解析快照失败", "err", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 已认证的请求：请求体里写的集群也必须是令牌的集群
	identity := identityFrom(r)
	if identity.authenticated && snapshot.ClusterID != "" && snapshot.ClusterID != identity.clusterID {
		log.Warn("快照里的集群与令牌所属集群不一致，已拒绝",
			"tokenCluster", identity.clusterID, "claimed", snapshot.ClusterID)
		http.Error(w, "token does not belong to the claimed cluster", http.StatusForbidden)
		return
	}

	// 未认证时沿用自报的集群：优先请求头，其次请求体
	selfReported := r.Header.Get("X-Cluster-ID")
	if selfReported == "" {
		selfReported = snapshot.ClusterID
	}
	clusterID := clusterFor(r, selfReported)

	if clusterID == "" {
		http.Error(w, "cluster_id is required", http.StatusBadRequest)
		return
	}

	// 确保 snapshot 中的 ClusterID 与 Header 一致
	snapshot.ClusterID = clusterID

	// 通过 Processor 处理
	if err := s.processor.ProcessSnapshot(clusterID, &snapshot); err != nil {
		log.Error("处理器错误", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	s.authStatus.record(clusterID, identity.authenticated)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) snapshotLimit() int64 {
	if s.maxSnapshotBytes > 0 {
		return s.maxSnapshotBytes
	}
	return defaultMaxSnapshotBytes
}

var errSnapshotTooLarge = errors.New("snapshot exceeds size limit")

// cappedReader 限制解压后能读出的字节数（http.MaxBytesReader 只管压缩前的大小）
type cappedReader struct {
	source    io.Reader
	remaining int64
}

func (c *cappedReader) Read(p []byte) (int, error) {
	if c.remaining <= 0 {
		return 0, errSnapshotTooLarge
	}
	if int64(len(p)) > c.remaining {
		p = p[:c.remaining]
	}
	n, err := c.source.Read(p)
	c.remaining -= int64(n)
	return n, err
}

func isBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.Is(err, errSnapshotTooLarge) || errors.As(err, &tooLarge)
}
