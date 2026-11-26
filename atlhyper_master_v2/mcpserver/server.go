package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/go-chi/httprate"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/auth/apitoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/httpserve"
	"AtlHyper/common/logger"
	"AtlHyper/common/truncate"
)

var log = logger.Module("MCP")

const (
	// endpointPath MCP 的 HTTP 路径
	endpointPath = "/mcp"

	defaultRateLimitPerMinute = 120
	defaultMaxResultRunes     = 20000
	defaultMaxBodyBytes       = 256 << 10
	defaultCallTimeout        = 45 * time.Second

	// preAuthRequestsPerMinute 校验令牌之前按对端地址的限流：挡住不带有效令牌的大量请求
	preAuthRequestsPerMinute = 600

	// 审计里的固定取值
	auditSource   = "mcp"
	auditAction   = "execute"
	auditResource = "mcp_tool"
	// maxAuditBodyRunes 审计里记录的调用参数上限
	maxAuditBodyRunes = 2000

	// callerExtraKey 调用者在令牌信息附加字段里的键
	callerExtraKey = "caller"
	// remoteAddrExtraKey 对端地址在令牌信息附加字段里的键（工具处理函数拿不到 *http.Request）
	remoteAddrExtraKey = "remoteAddr"
)

// instructions 给客户端模型看的用法说明
const instructions = "AtlHyper 的只读运维查询。想了解集群整体状况先用 get_cluster_overview 或 get_cluster_risk，" +
	"再按线索用其他工具深入。有多个集群时每次调用都要带 cluster_id（用 list_clusters 查看）。" +
	"结果是 JSON 文本；调用失败时结果的 isError 为 true，元数据 atlhyper.io/errorCode 给出失败类别。"

// errVerifierUnavailable 暂时无法校验令牌时交给客户端的说明（原因只进日志）
var errVerifierUnavailable = errors.New("暂时无法验证身份，请稍后重试")

// Server MCP 入口
type Server struct {
	cfg        Config
	byLevel    map[int]*mcp.Server
	noTools    *mcp.Server // 等级不在已知范围内的调用者：什么工具也看不到
	handler    http.Handler
	httpServer *http.Server
}

// NewServer 创建 MCP 入口：按开放名单与各角色等级备好工具清单
func NewServer(cfg Config) (*Server, error) {
	return newServer(cfg, exposedTools)
}

func newServer(cfg Config, exposed []string) (*Server, error) {
	switch {
	case cfg.Tools == nil:
		return nil, errors.New("mcpserver: 缺少工具注册表")
	case cfg.Authenticator == nil:
		return nil, errors.New("mcpserver: 缺少令牌校验器")
	case cfg.Clusters == nil:
		return nil, errors.New("mcpserver: 缺少集群列表")
	case cfg.Audit == nil:
		return nil, errors.New("mcpserver: 缺少审计")
	case cfg.Allowed == nil:
		return nil, errors.New("mcpserver: 缺少权限判断")
	}
	if cfg.RateLimitPerMinute <= 0 {
		cfg.RateLimitPerMinute = defaultRateLimitPerMinute
	}
	if cfg.MaxResultRunes <= 0 {
		cfg.MaxResultRunes = defaultMaxResultRunes
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = defaultMaxBodyBytes
	}
	if cfg.CallTimeout <= 0 {
		cfg.CallTimeout = defaultCallTimeout
	}
	if cfg.Version == "" {
		cfg.Version = buildVersion()
	}

	tools, err := resolveExposedTools(cfg.Tools, exposed)
	if err != nil {
		return nil, fmt.Errorf("mcpserver: %w", err)
	}

	s := &Server{cfg: cfg, byLevel: make(map[int]*mcp.Server, len(cfg.Levels))}
	s.noTools = s.newProtocolServer()
	for _, level := range cfg.Levels {
		server := s.newProtocolServer()
		allow := s.allowFor(level)
		for _, t := range tools {
			if t.tool.VisibleTo(allow) {
				server.AddTool(t.definition(), s.callHandler(t.tool))
			}
		}
		s.byLevel[level] = server
	}
	s.handler = s.buildHandler()
	return s, nil
}

func (s *Server) newProtocolServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: "atlhyper", Version: s.cfg.Version},
		&mcp.ServerOptions{Instructions: instructions})
}

func (s *Server) allowFor(level int) toolkit.PermissionCheck {
	return func(permission string) bool { return s.cfg.Allowed(level, permission) }
}

// Handler 入口的 HTTP 处理器。
// 顺序：拒绝浏览器跨站请求 → 按对端地址限流 → 令牌校验 → 按令牌限流 → MCP 协议处理（无状态，JSON 响应，请求体有上限）
func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) buildHandler() http.Handler {
	protocol := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		caller, ok := callerFromTokenInfo(auth.TokenInfoFromContext(r.Context()))
		if !ok {
			return s.noTools
		}
		if server, known := s.byLevel[caller.Level]; known {
			return server
		}
		return s.noTools
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: s.cfg.MaxBodyBytes})

	perToken := httprate.Limit(s.cfg.RateLimitPerMinute, time.Minute,
		httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
			caller, _ := callerFromTokenInfo(auth.TokenInfoFromContext(r.Context()))
			return "token:" + strconv.FormatInt(caller.TokenID, 10) + ":" + strconv.FormatInt(caller.UserID, 10), nil
		}),
		httprate.WithLimitHandler(tooManyRequests),
	)
	perPeer := httprate.Limit(preAuthRequestsPerMinute, time.Minute,
		httprate.WithKeyFuncs(func(r *http.Request) (string, error) { return remoteHost(r), nil }),
		httprate.WithLimitHandler(tooManyRequests),
	)
	bearer := auth.RequireBearerToken(s.verifyToken, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true})

	// 没有浏览器客户端：带着别处 Origin 的请求（浏览器里的跨站脚本、DNS 重绑定）一律拒绝
	crossOrigin := http.NewCrossOriginProtection()

	mux := http.NewServeMux()
	mux.Handle(endpointPath, crossOrigin.Handler(perPeer(bearer(perToken(protocol)))))
	return mux
}

func tooManyRequests(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"error":"请求过于频繁，请稍后再试"}`))
}

func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// verifyToken 令牌 → 调用者。无效的令牌回 401；暂时无法校验回 5xx，原因只进日志
func (s *Server) verifyToken(ctx context.Context, token string, r *http.Request) (*auth.TokenInfo, error) {
	caller, err := s.cfg.Authenticator.Authenticate(ctx, token)
	if err != nil {
		if errors.Is(err, apitoken.ErrInvalidToken) {
			return nil, auth.ErrInvalidToken
		}
		log.Error("校验访问令牌失败", "err", err)
		return nil, errVerifierUnavailable
	}
	return &auth.TokenInfo{
		UserID:     strconv.FormatInt(caller.UserID, 10),
		Expiration: caller.ExpiresAt,
		Extra:      map[string]any{callerExtraKey: caller, remoteAddrExtraKey: remoteHost(r)},
	}, nil
}

func callerFromTokenInfo(info *auth.TokenInfo) (apitoken.Caller, bool) {
	if info == nil {
		return apitoken.Caller{}, false
	}
	caller, ok := info.Extra[callerExtraKey].(apitoken.Caller)
	return caller, ok
}

// callHandler 一个工具的 MCP 处理函数：确定集群 → 按调用者当前等级再查权限并执行 → 结果与审计
func (s *Server) callHandler(tool *toolkit.Tool) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		started := time.Now()
		caller, ok := callerFromTokenInfo(req.Extra.TokenInfo)
		if !ok {
			// 入口顺序保证了走到这里一定有调用者；没有就拒绝，不按匿名执行
			return failureResult(CodePermissionDenied, "未识别出调用者"), nil
		}

		clusterID, toolArgs, output, err := s.execute(ctx, tool, caller, req.Params.Arguments)

		var result *mcp.CallToolResult
		code := ""
		if err != nil {
			var message string
			code, message = classifyFailure(err)
			log.Warn("工具调用失败", "tool", tool.Name(), "caller", caller.Username, "cluster", clusterID, "code", code, "err", err)
			result = failureResult(code, message)
		} else {
			result = s.successResult(output)
		}
		s.recordAudit(ctx, req, caller, tool.Name(), clusterID, toolArgs, code, started)
		return result, nil
	}
}

func (s *Server) execute(ctx context.Context, tool *toolkit.Tool, caller apitoken.Caller, arguments json.RawMessage) (clusterID string, toolArgs []byte, output string, err error) {
	toolArgs, requested, err := splitClusterArgument(arguments, tool.NeedsCluster())
	if err != nil {
		return "", nil, "", err
	}
	if tool.NeedsCluster() {
		if clusterID, err = s.resolveCluster(ctx, requested); err != nil {
			return "", toolArgs, "", err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, s.cfg.CallTimeout)
	defer cancel()
	output, err = s.cfg.Tools.Execute(ctx, clusterID, s.allowFor(caller.Level), &llm.ToolCall{Name: tool.Name(), Params: string(toolArgs)})
	return clusterID, toolArgs, output, err
}

// recordAudit 每次工具调用记一行。写审计失败不影响调用结果，但要留痕
func (s *Server) recordAudit(ctx context.Context, req *mcp.CallToolRequest, caller apitoken.Caller, toolName, clusterID string, toolArgs []byte, failureCode string, started time.Time) {
	body, _ := json.Marshal(struct {
		Cluster   string          `json:"cluster,omitempty"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}{Cluster: clusterID, Arguments: toolArgs})

	entry := &database.AuditLog{
		Timestamp:    started,
		UserID:       caller.UserID,
		Username:     caller.Username,
		Role:         caller.Level,
		Source:       auditSource,
		Action:       auditAction,
		Resource:     auditResource,
		Method:       "MCP",
		Path:         "tools/call/" + toolName,
		RequestBody:  truncate.Runes(string(body), maxAuditBodyRunes, "…"),
		StatusCode:   auditStatus(failureCode),
		Success:      failureCode == "",
		ErrorMessage: failureCode,
		DurationMs:   time.Since(started).Milliseconds(),
	}
	if req.Extra != nil {
		entry.UserAgent = truncate.Runes(req.Extra.Header.Get("User-Agent"), 200, "…")
		if req.Extra.TokenInfo != nil {
			entry.IP, _ = req.Extra.TokenInfo.Extra[remoteAddrExtraKey].(string)
		}
	}
	// 调用方断开不该让这一行丢掉
	if err := s.cfg.Audit.Create(context.WithoutCancel(ctx), entry); err != nil {
		log.Warn("写审计失败", "tool", toolName, "caller", caller.Username, "err", err)
	}
}

// auditStatus 审计里的状态码：按失败类别给出对应的 HTTP 语义，便于在审计页筛选
func auditStatus(failureCode string) int {
	switch failureCode {
	case "":
		return http.StatusOK
	case CodePermissionDenied:
		return http.StatusForbidden
	case CodeInvalidParams, CodeClusterRequired, CodeClusterNotFound, CodeNoCluster:
		return http.StatusBadRequest
	case CodeTimeout:
		return http.StatusGatewayTimeout
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Start 开始监听（确认端口监听成功后返回）
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.cfg.Port),
		Handler:           s.handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		// 响应是一次性的 JSON：写超时只需盖过单次工具调用的时限
		WriteTimeout: s.cfg.CallTimeout + 15*time.Second,
	}
	if err := httpserve.Start(s.httpServer, func(err error) { log.Error("服务器错误", "err", err) }); err != nil {
		return fmt.Errorf("启动 MCP 入口: %w", err)
	}
	log.Info("启动服务器", "port", s.cfg.Port, "path", endpointPath)
	return nil
}

// Stop 停止监听
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// buildVersion 告诉客户端的服务端版本：构建信息里的模块版本，没有时取代码提交的前 12 位，都取不到为 unknown
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	if version := info.Main.Version; version != "" && version != "(devel)" {
		return version
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 12 {
			return setting.Value[:12]
		}
	}
	return "unknown"
}
