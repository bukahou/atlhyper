// atlhyper_master_v2/agentsdk/auth.go
// Agent 身份认证：由请求携带的集群令牌确定它属于哪个集群
package agentsdk

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/agentsdk/clustertoken"
	"AtlHyper/atlhyper_master_v2/database"
)

// AuthMode Agent 入口的认证模式
type AuthMode string

const (
	// AuthModeOff 不检查令牌
	AuthModeOff AuthMode = "off"
	// AuthModeWarn 没带有效令牌的请求照常接收，但记录告警。用于升级期间观察哪些集群还没配令牌
	AuthModeWarn AuthMode = "warn"
	// AuthModeEnforce 没带有效令牌的请求一律拒绝
	AuthModeEnforce AuthMode = "enforce"
)

// ParseAuthMode 解析配置里的认证模式。未知的值报错，不悄悄落到某一档
func ParseAuthMode(value string) (AuthMode, error) {
	switch mode := AuthMode(value); mode {
	case AuthModeOff, AuthModeWarn, AuthModeEnforce:
		return mode, nil
	}
	return "", fmt.Errorf("未知的 Agent 认证模式 %q（可选 off / warn / enforce）", value)
}

// ClusterDirectory 按令牌哈希查集群（集群注册表的最小读接口）
type ClusterDirectory interface {
	GetByTokenHash(ctx context.Context, tokenHash string) (*database.Cluster, error)
}

// agentIdentity 一个请求经认证后的身份
type agentIdentity struct {
	clusterID     string // 令牌所属的集群；未认证时为空
	authenticated bool
}

type agentIdentityKey struct{}

// identityFrom 取出中间件放进请求里的身份；没经过中间件时返回未认证的空身份
func identityFrom(r *http.Request) agentIdentity {
	identity, _ := r.Context().Value(agentIdentityKey{}).(agentIdentity)
	return identity
}

// clusterFor 这个请求应当记到哪个集群名下：已认证时一律用令牌的集群，否则用请求自报的
func clusterFor(r *http.Request, selfReported string) string {
	if identity := identityFrom(r); identity.authenticated {
		return identity.clusterID
	}
	return selfReported
}

// authenticateAgent 认证中间件
func (s *Server) authenticateAgent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, rejection := s.resolveIdentity(r)
		if rejection != nil {
			http.Error(w, rejection.message, rejection.status)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), agentIdentityKey{}, identity)))
	}
}

type authRejection struct {
	status  int
	message string
}

// resolveIdentity 按认证模式确定请求的身份，或给出拒绝理由
func (s *Server) resolveIdentity(r *http.Request) (agentIdentity, *authRejection) {
	if s.authMode == AuthModeOff || s.authMode == "" {
		return agentIdentity{}, nil
	}
	enforce := s.authMode == AuthModeEnforce
	selfReported := selfReportedClusterIDs(r)

	token := clustertoken.FromBearerHeader(r.Header.Get("Authorization"))
	var registered *database.Cluster
	reason := "没带令牌"
	if token != "" {
		found, err := s.clusters.GetByTokenHash(r.Context(), clustertoken.Hash(token))
		switch {
		case err != nil:
			// 查不了注册表时不能当作通过：强制模式下拒绝，让 Agent 稍后重试
			if enforce {
				log.Error("查询集群注册表失败，拒绝请求", "path", r.URL.Path, "err", err)
				return agentIdentity{}, &authRejection{http.StatusServiceUnavailable, "cluster registry unavailable"}
			}
			reason = "注册表暂不可用"
		case found != nil && clustertoken.Matches(token, found.TokenHash):
			registered = found
		default:
			reason = "令牌无效"
		}
	}

	if registered == nil {
		if enforce {
			return agentIdentity{}, &authRejection{http.StatusUnauthorized, "valid cluster token required"}
		}
		s.warnUnauthenticated(firstNonEmpty(selfReported), reason)
		return agentIdentity{}, nil
	}

	for _, claimed := range selfReported {
		if claimed != registered.ClusterID {
			log.Warn("令牌所属集群与请求自报的集群不一致，已拒绝",
				"tokenCluster", registered.ClusterID, "claimed", claimed, "path", r.URL.Path)
			return agentIdentity{}, &authRejection{http.StatusForbidden, "token does not belong to the claimed cluster"}
		}
	}
	return agentIdentity{clusterID: registered.ClusterID, authenticated: true}, nil
}

// selfReportedClusterIDs 请求头与查询参数里自报的集群 ID（去掉空值）。
// 请求体里的集群 ID（快照）由处理函数自己比对。
func selfReportedClusterIDs(r *http.Request) []string {
	var claimed []string
	for _, value := range []string{r.Header.Get("X-Cluster-ID"), r.URL.Query().Get("cluster_id")} {
		if value != "" {
			claimed = append(claimed, value)
		}
	}
	return claimed
}

func firstNonEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// warnUnauthenticated 告警模式下记录没带有效令牌的请求（按集群与原因限频）
func (s *Server) warnUnauthenticated(clusterID, reason string) {
	if s.warnings == nil || !s.warnings.allow(clusterID+"|"+reason) {
		return
	}
	log.Warn("Agent 请求没有通过令牌认证，当前为告警模式，已照常接收",
		"cluster", clusterID, "reason", reason)
}

// ==================== 告警限频 ====================

// warnLimiterMaxKeys 记录的键数上限；超过后清空重来，防止任意集群 ID 把表撑大
const warnLimiterMaxKeys = 1024

// warnLimiter 同一个键在一个时间窗内只放行一次
type warnLimiter struct {
	mu     sync.Mutex
	window time.Duration
	now    func() time.Time
	last   map[string]time.Time
}

func newWarnLimiter(window time.Duration, now func() time.Time) *warnLimiter {
	return &warnLimiter{window: window, now: now, last: make(map[string]time.Time)}
}

func (l *warnLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.now()
	if previous, ok := l.last[key]; ok && current.Sub(previous) < l.window {
		return false
	}
	if len(l.last) >= warnLimiterMaxKeys {
		l.last = make(map[string]time.Time)
	}
	l.last[key] = current
	return true
}

// ==================== 各集群的认证状态 ====================

// AuthStatusTracker 记录每个集群最近一次上报是否带了有效令牌，供管理页显示
type AuthStatusTracker struct {
	mu            sync.RWMutex
	authenticated map[string]bool
}

// NewAuthStatusTracker 创建认证状态记录器
func NewAuthStatusTracker() *AuthStatusTracker {
	return &AuthStatusTracker{authenticated: make(map[string]bool)}
}

func (t *AuthStatusTracker) record(clusterID string, authenticated bool) {
	if t == nil || clusterID == "" {
		return
	}
	t.mu.Lock()
	t.authenticated[clusterID] = authenticated
	t.mu.Unlock()
}

// IsAuthenticated 该集群最近一次上报是否带了有效令牌；known 为假表示它还没上报过
func (t *AuthStatusTracker) IsAuthenticated(clusterID string) (authenticated, known bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	authenticated, known = t.authenticated[clusterID]
	return authenticated, known
}
