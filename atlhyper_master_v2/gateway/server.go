// atlhyper_master_v2/gateway/server.go
// Gateway HTTP Server（Web API）
// Gateway 是外部访问层，禁止直接访问 DataHub
// 读取通过 Query 层，写入通过 CommandService
//
// 路由注册见 routes.go
package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/query"
	"AtlHyper/atlhyper_master_v2/service"
)

// Server Gateway HTTP Server
type Server struct {
	port           int
	query          query.Query
	commandService service.CommandService
	database       database.Database
	datahub        datahub.DataHub
	httpServer     *http.Server
}

// Config Server 配置
type Config struct {
	Port           int
	Query          query.Query
	CommandService service.CommandService
	Database       database.Database
	DataHub        datahub.DataHub
}

// NewServer 创建 Server
func NewServer(cfg Config) *Server {
	return &Server{
		port:           cfg.Port,
		query:          cfg.Query,
		commandService: cfg.CommandService,
		database:       cfg.Database,
		datahub:        cfg.DataHub,
	}
}

// Start 启动 Server
func (s *Server) Start() error {
	// 使用 Router 统一管理路由（见 routes.go）
	router := NewRouter(s.query, s.commandService, s.database, s.datahub)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      router.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("[Gateway] 启动服务器: 端口=%d", s.port)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Gateway] 服务器错误: %v", err)
		}
	}()

	return nil
}

// Stop 停止 Server
func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
