// atlhyper_master_v2/agentsdk/server.go
// AgentSDK HTTP Server
// 负责接收 Agent 的请求（快照、心跳、执行结果）和下发指令
// 数据处理通过 Processor 层，不直接访问 DataHub
package agentsdk

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/processor"
	"AtlHyper/common/logger"
)

var log = logger.Module("AgentSDK")

// Server AgentSDK HTTP Server
type Server struct {
	port       int
	timeout    time.Duration
	bus        mq.Consumer
	processor  processor.Processor
	cmdRepo    database.CommandHistoryRepository
	cmdTTL     time.Duration // 指令有效期，0 表示用默认值
	httpServer *http.Server

	authMode         AuthMode
	clusters         ClusterDirectory
	warnings         *warnLimiter
	authStatus       *AuthStatusTracker
	maxSnapshotBytes int64 // 快照压缩前后的大小上限
}

const (
	// defaultMaxSnapshotBytes 快照大小上限的默认值
	defaultMaxSnapshotBytes = 256 << 20
	// unauthenticatedWarnWindow 同一集群同一原因的「未认证」告警最短间隔
	unauthenticatedWarnWindow = time.Minute
)

// Config Server 配置
type Config struct {
	Port           int
	CommandTimeout time.Duration
	Bus            mq.Consumer
	Processor      processor.Processor
	CmdRepo        database.CommandHistoryRepository

	// AuthMode 认证模式；空值等同 off
	AuthMode AuthMode
	// Clusters 集群注册表；AuthMode 不是 off 时必须提供
	Clusters ClusterDirectory
	// AuthStatus 记录各集群是否已带令牌（可选，供管理页显示）
	AuthStatus *AuthStatusTracker
	// MaxSnapshotBytes 快照大小上限；0 表示用默认值
	MaxSnapshotBytes int64
}

// NewServer 创建 Server
func NewServer(cfg Config) *Server {
	maxSnapshotBytes := cfg.MaxSnapshotBytes
	if maxSnapshotBytes <= 0 {
		maxSnapshotBytes = defaultMaxSnapshotBytes
	}
	return &Server{
		port:             cfg.Port,
		timeout:          cfg.CommandTimeout,
		bus:              cfg.Bus,
		processor:        cfg.Processor,
		cmdRepo:          cfg.CmdRepo,
		authMode:         cfg.AuthMode,
		clusters:         cfg.Clusters,
		warnings:         newWarnLimiter(unauthenticatedWarnWindow, time.Now),
		authStatus:       cfg.AuthStatus,
		maxSnapshotBytes: maxSnapshotBytes,
	}
}

// routes 注册全部入口。四条 Agent 通道都先经过认证中间件；健康检查不需要令牌。
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/agent/snapshot", s.authenticateAgent(s.handleSnapshot))
	mux.HandleFunc("/agent/heartbeat", s.authenticateAgent(s.handleHeartbeat))
	mux.HandleFunc("/agent/commands", s.authenticateAgent(s.handleCommands))
	mux.HandleFunc("/agent/result", s.authenticateAgent(s.handleResult))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	return mux
}

// Start 启动 Server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: s.timeout + 10*time.Second, // 长轮询需要更长的写超时
	}

	log.Info("启动服务器", "port", s.port, "authMode", s.authMode)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("服务器错误", "err", err)
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
