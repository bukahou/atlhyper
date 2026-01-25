// service/ingest/service.go
package ingest

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/model/transport"
)

// Service 定义 Ingest 服务接口
// -----------------------------------------------------------------------------
// 职责：
//   - 接收 Agent 上报的数据
//   - 进行数据校验
//   - 调用 Repository 层写入 DataHub
// -----------------------------------------------------------------------------
type Service interface {
	// 快照型数据处理（ReplaceLatest 语义）
	ProcessPodList(ctx context.Context, env transport.Envelope) error
	ProcessNodeList(ctx context.Context, env transport.Envelope) error
	ProcessServiceList(ctx context.Context, env transport.Envelope) error
	ProcessNamespaceList(ctx context.Context, env transport.Envelope) error
	ProcessIngressList(ctx context.Context, env transport.Envelope) error
	ProcessDeploymentList(ctx context.Context, env transport.Envelope) error
	ProcessConfigMapList(ctx context.Context, env transport.Envelope) error
	ProcessMetricsSnapshot(ctx context.Context, env transport.Envelope) error

	// 增量型数据处理（Append 语义）
	ProcessEvents(ctx context.Context, env transport.Envelope) error
	ProcessEventsBatch(ctx context.Context, envs []transport.Envelope) error
}

// -----------------------------------------------------------------------------
// service 实现
// -----------------------------------------------------------------------------
type service struct {
	validator *Validator
}

// New 创建 Ingest Service 实例
func New() Service {
	return &service{
		validator: NewValidator(),
	}
}

// 全局单例
var defaultService Service

// Init 初始化全局 Ingest Service
func Init() {
	defaultService = New()
}

// Default 获取全局 Ingest Service 实例
func Default() Service {
	if defaultService == nil {
		defaultService = New()
	}
	return defaultService
}

// -----------------------------------------------------------------------------
// 快照型数据处理（每种资源类型只保留最新一份）
// -----------------------------------------------------------------------------

func (s *service) ProcessPodList(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourcePodListSnapshot); err != nil {
		return fmt.Errorf("validate pod list: %w", err)
	}
	return repository.MemW.ReplaceLatest(ctx, env)
}

func (s *service) ProcessNodeList(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourceNodeListSnapshot); err != nil {
		return fmt.Errorf("validate node list: %w", err)
	}
	return repository.MemW.ReplaceLatest(ctx, env)
}

func (s *service) ProcessServiceList(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourceServiceListSnapshot); err != nil {
		return fmt.Errorf("validate service list: %w", err)
	}
	return repository.MemW.ReplaceLatest(ctx, env)
}

func (s *service) ProcessNamespaceList(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourceNamespaceListSnapshot); err != nil {
		return fmt.Errorf("validate namespace list: %w", err)
	}
	return repository.MemW.ReplaceLatest(ctx, env)
}

func (s *service) ProcessIngressList(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourceIngressListSnapshot); err != nil {
		return fmt.Errorf("validate ingress list: %w", err)
	}
	return repository.MemW.ReplaceLatest(ctx, env)
}

func (s *service) ProcessDeploymentList(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourceDeploymentListSnapshot); err != nil {
		return fmt.Errorf("validate deployment list: %w", err)
	}
	return repository.MemW.ReplaceLatest(ctx, env)
}

func (s *service) ProcessConfigMapList(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourceConfigMapListSnapshot); err != nil {
		return fmt.Errorf("validate configmap list: %w", err)
	}
	return repository.MemW.ReplaceLatest(ctx, env)
}

func (s *service) ProcessMetricsSnapshot(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourceMetricsSnapshot); err != nil {
		return fmt.Errorf("validate metrics snapshot: %w", err)
	}
	return repository.MemW.ReplaceLatest(ctx, env)
}

// -----------------------------------------------------------------------------
// 增量型数据处理（追加模式）
// -----------------------------------------------------------------------------

func (s *service) ProcessEvents(ctx context.Context, env transport.Envelope) error {
	if err := s.validator.ValidateEnvelope(env, transport.SourceK8sEvent); err != nil {
		return fmt.Errorf("validate events: %w", err)
	}
	return repository.MemW.AppendEnvelope(ctx, env)
}

func (s *service) ProcessEventsBatch(ctx context.Context, envs []transport.Envelope) error {
	for _, env := range envs {
		if err := s.validator.ValidateEnvelope(env, transport.SourceK8sEvent); err != nil {
			return fmt.Errorf("validate events batch: %w", err)
		}
	}
	return repository.MemW.AppendEnvelopeBatch(ctx, envs)
}
