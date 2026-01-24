// atlhyper_master_v2/agentsdk/server.go
// AgentSDK HTTP Server
// 负责接收 Agent 的请求（快照、心跳、执行结果）和下发指令
// 数据处理通过 Processor 层，不直接访问 DataHub
package agentsdk

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/processor"
)

// Server AgentSDK HTTP Server
type Server struct {
	port       int
	timeout    time.Duration
	bus        mq.CommandBus
	processor  processor.Processor
	httpServer *http.Server
}

// Config Server 配置
type Config struct {
	Port           int
	CommandTimeout time.Duration
	Bus            mq.CommandBus
	Processor      processor.Processor
}

// NewServer 创建 Server
func NewServer(cfg Config) *Server {
	return &Server{
		port:      cfg.Port,
		timeout:   cfg.CommandTimeout,
		bus:       cfg.Bus,
		processor: cfg.Processor,
	}
}

// Start 启动 Server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// 注册路由
	mux.HandleFunc("/agent/snapshot", s.handleSnapshot)
	mux.HandleFunc("/agent/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("/agent/commands", s.handleCommands)
	mux.HandleFunc("/agent/result", s.handleResult)

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: s.timeout + 10*time.Second, // 长轮询需要更长的写超时
	}

	log.Printf("[AgentSDK] 启动服务器: 端口=%d", s.port)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[AgentSDK] 服务器错误: %v", err)
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
