// atlhyper_master_v2/tester/server.go
// 测试服务器
// 独立端口运行，与业务 Gateway 完全隔离
package tester

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/notifier"
)

// Server 测试服务器
type Server struct {
	port       int
	registry   *Registry
	handler    *Handler
	httpServer *http.Server
}

// Config 服务器配置
type Config struct {
	Port         int
	AlertManager *notifier.AlertManager
	// 未来可添加更多依赖
	// Store datahub.Store
	// AIService ai.AIService
}

// NewServer 创建测试服务器
func NewServer(cfg Config) *Server {
	// 创建注册表
	registry := NewRegistry()

	// 注册测试器
	if cfg.AlertManager != nil {
		registry.Register(NewNotifierTester(cfg.AlertManager))
	}

	// 创建 Handler
	handler := NewHandler(registry)

	return &Server{
		port:     cfg.Port,
		registry: registry,
		handler:  handler,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// 注册路由
	mux.HandleFunc("/health", s.handler.Health)
	mux.HandleFunc("/testers", s.handler.List)
	mux.HandleFunc("/test/", s.handler.Test)

	// CORS 中间件
	handler := s.corsMiddleware(mux)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Printf("[Tester] 启动测试服务器: 端口=%d", s.port)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Tester] 服务器错误: %v", err)
		}
	}()

	return nil
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		log.Println("[Tester] 停止测试服务器")
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// corsMiddleware CORS 中间件
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Registry 获取注册表（用于外部注册更多测试器）
func (s *Server) Registry() *Registry {
	return s.registry
}
