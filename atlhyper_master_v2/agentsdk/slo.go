// Package agentsdk Agent 通信层
//
// slo.go - SLO 端点（已废弃）
//
// SLO 数据已迁移到 ClusterSnapshot.SLOData，随快照统一推送。
// 独立的 /agent/slo 端点已废弃，Master P2 会完全移除。
package agentsdk

import (
	"net/http"
)

// handleSLO 处理 SLO 指标推送（已废弃）
//
// SLO 数据现在通过 ClusterSnapshot.SLOData 随快照统一推送。
// 保留此 handler 以兼容旧版 Agent，返回 410 Gone。
func (s *Server) handleSLO(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "SLO endpoint deprecated, use ClusterSnapshot.SLOData", http.StatusGone)
}
