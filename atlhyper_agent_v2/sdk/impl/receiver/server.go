// Package receiver 通用数据接收服务器
//
// 本包提供 HTTP 服务器，被动接收外部组件推送的数据并暂存于内存。
// Repository 层通过接口方法拉取数据，与主动拉取型 SDK 调用姿势一致。
//
// 当前支持:
//   - /metrics/node — 接收节点指标推送
//   - /health       — 健康检查
package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var log = logger.Module("Receiver")

// Server 数据接收服务器，实现 sdk.ReceiverClient
type Server struct {
	port       int
	httpServer *http.Server

	// 节点指标内存缓存（按节点名覆盖式存储）
	mu          sync.RWMutex
	nodeMetrics map[string]*model_v2.NodeMetricsSnapshot
}

// NewServer 创建数据接收服务器
func NewServer(port int) sdk.ReceiverClient {
	s := &Server{
		port:        port,
		nodeMetrics: make(map[string]*model_v2.NodeMetricsSnapshot),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics/node", s.handleNodeMetrics)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

// =============================================================================
// sdk.ReceiverClient 接口实现
// =============================================================================

// Start 启动 HTTP 服务器
func (s *Server) Start() error {
	go func() {
		log.Info("Receiver 启动", "port", s.port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Receiver 启动失败", "err", err)
		}
	}()
	return nil
}

// Stop 停止 HTTP 服务器
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown receiver: %w", err)
	}

	log.Info("Receiver 已停止")
	return nil
}

// GetAllNodeMetrics 获取所有节点指标（返回副本）
func (s *Server) GetAllNodeMetrics() map[string]*model_v2.NodeMetricsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*model_v2.NodeMetricsSnapshot, len(s.nodeMetrics))
	for k, v := range s.nodeMetrics {
		result[k] = v
	}
	return result
}

// =============================================================================
// HTTP 处理
// =============================================================================

// handleNodeMetrics 处理节点指标上报
// POST /metrics/node
func (s *Server) handleNodeMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var snapshot model_v2.NodeMetricsSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if snapshot.NodeName == "" {
		http.Error(w, "node_name is required", http.StatusBadRequest)
		return
	}

	// 写入内存缓存
	s.mu.Lock()
	s.nodeMetrics[snapshot.NodeName] = &snapshot
	s.mu.Unlock()

	log.Debug("收到节点指标",
		"node", snapshot.NodeName,
		"cpu", fmt.Sprintf("%.1f%%", snapshot.CPU.UsagePercent),
		"mem", fmt.Sprintf("%.1f%%", snapshot.Memory.UsagePercent),
	)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleHealth 健康检查
// GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
