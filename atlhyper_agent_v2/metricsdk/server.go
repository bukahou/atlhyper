// Package metricsdk 节点指标 SDK
//
// 本包提供 HTTP 服务器，用于:
//   - 接收 atlhyper_metrics_v2 推送的节点指标
//   - 存储到 MetricsRepository
package metricsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var log = logger.Module("MetricsSDK")

// Server Metrics SDK HTTP 服务器
type Server struct {
	port        int
	metricsRepo repository.MetricsRepository
	httpServer  *http.Server
}

// Config 服务器配置
type Config struct {
	Port        int
	MetricsRepo repository.MetricsRepository
}

// NewServer 创建 Metrics SDK 服务器
func NewServer(cfg Config) *Server {
	s := &Server{
		port:        cfg.Port,
		metricsRepo: cfg.MetricsRepo,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics/node", s.handleNodeMetrics)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

// Start 启动服务器
func (s *Server) Start() error {
	go func() {
		log.Info("Metrics SDK 启动", "port", s.port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Metrics SDK 启动失败", "err", err)
		}
	}()
	return nil
}

// Stop 停止服务器
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown metrics sdk: %w", err)
	}

	log.Info("Metrics SDK 已停止")
	return nil
}

// handleNodeMetrics 处理节点指标上报
// POST /metrics/node
func (s *Server) handleNodeMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 限制请求体大小 (1MB)
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

	// 保存到 Repository
	s.metricsRepo.Save(&snapshot)

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
