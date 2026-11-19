// atlhyper_master_v2/agentsdk/server.go
// AgentSDK HTTP Server
// 负责接收 Agent 的请求（快照、心跳、执行结果）和下发指令
// 数据处理通过 Processor 层，不直接访问 DataHub
package agentsdk

import (
	"AtlHyper/model_v3/agentproto"
	"context"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/processor"
	"AtlHyper/common/httpserve"
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
		authMode:         authModeOrEnforce(cfg.AuthMode),
		clusters:         cfg.Clusters,
		warnings:         newWarnLimiter(unauthenticatedWarnWindow, time.Now),
		authStatus:       cfg.AuthStatus,
		maxSnapshotBytes: maxSnapshotBytes,
	}
}

// routes 注册全部入口。四条 Agent 通道都先经过认证中间件；健康检查不需要令牌。
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/agent/snapshot", requireProtocol(s.authenticateAgent(s.handleSnapshot)))
	mux.HandleFunc("/agent/heartbeat", requireProtocol(s.authenticateAgent(s.handleHeartbeat)))
	mux.HandleFunc("/agent/commands", requireProtocol(s.authenticateAgent(s.handleCommands)))
	mux.HandleFunc("/agent/result", requireProtocol(s.authenticateAgent(s.handleResult)))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	return mux
}

// Start 启动 Server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      s.timeout + 10*time.Second, // 长轮询需要更长的写超时
	}

	if err := httpserve.Start(s.httpServer, func(err error) { log.Error("服务器错误", "err", err) }); err != nil {
		return fmt.Errorf("启动 Agent 入口: %w", err)
	}
	log.Info("启动服务器", "port", s.port, "authMode", s.authMode)
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

// requireProtocol 只接受当前协议版本的代理请求：不带或版本不对 → 426，⛔ 不按零值继续处理。
// 先于认证执行：版本对不上时连请求体都不读
func requireProtocol(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get(agentproto.Header)
		if !agentproto.Accepts(got) {
			log.Warn("代理协议版本不符，拒绝请求", "path", r.URL.Path, "agentVersion", got, "masterVersion", agentproto.Version,
				"cluster", r.Header.Get("X-Cluster-ID"))
			http.Error(w, fmt.Sprintf("agent protocol %q not supported, master speaks %d", got, agentproto.Version), http.StatusUpgradeRequired)
			return
		}
		next(w, r)
	}
}
