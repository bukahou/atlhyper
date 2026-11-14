// atlhyper_master_v2/service/operations/cluster_registry.go
// OpsClusterRegistry 实现 — 集群注册表的写入：登记、改显示名与描述、签发令牌、删除登记
package operations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/agentsdk/clustertoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/logger"
)

var clusterRegistryLog = logger.Module("ClusterRegistry")

// ClusterRegistryWriter 集群注册表的写入部分（读取在 service/query）
type ClusterRegistryWriter interface {
	Create(ctx context.Context, cluster *database.Cluster) error
	UpdateProfile(ctx context.Context, clusterID, name, description string) error
	SetToken(ctx context.Context, clusterID, tokenHash, tokenPrefix string, issuedAt time.Time) error
	Delete(ctx context.Context, clusterID string) error
}

// ClusterRegistryService 集群注册表写入操作
type ClusterRegistryService struct {
	clusterRepo ClusterRegistryWriter
	// 令牌生成与时钟可替换，便于测试
	generateToken func() (clustertoken.Issued, error)
	now           func() time.Time
}

// NewClusterRegistryService 创建 ClusterRegistryService
func NewClusterRegistryService(clusterRepo ClusterRegistryWriter) *ClusterRegistryService {
	return &ClusterRegistryService{
		clusterRepo:   clusterRepo,
		generateToken: clustertoken.Generate,
		now:           time.Now,
	}
}

// RegisterCluster 登记一个集群。新登记的集群没有令牌，需另行签发。
//
// 输入不合规 → model.ErrInvalidClusterInput；集群 ID 已登记 → database.ErrClusterExists
func (s *ClusterRegistryService) RegisterCluster(ctx context.Context, req *model.RegisterClusterRequest) error {
	if err := model.ValidateClusterID(req.ClusterID); err != nil {
		return err
	}
	name, description := strings.TrimSpace(req.Name), strings.TrimSpace(req.Description)
	if err := model.ValidateClusterProfile(name, description); err != nil {
		return err
	}

	cluster := &database.Cluster{ClusterID: req.ClusterID, Name: name, Description: description}
	if err := s.clusterRepo.Create(ctx, cluster); err != nil {
		return err
	}
	clusterRegistryLog.Info("集群已登记", "cluster", req.ClusterID)
	return nil
}

// UpdateClusterProfile 修改显示名与描述。
//
// 输入不合规 → model.ErrInvalidClusterInput；集群未登记 → database.ErrClusterNotFound
func (s *ClusterRegistryService) UpdateClusterProfile(ctx context.Context, clusterID string, req *model.UpdateClusterProfileRequest) error {
	name, description := strings.TrimSpace(req.Name), strings.TrimSpace(req.Description)
	if err := model.ValidateClusterProfile(name, description); err != nil {
		return err
	}
	return s.clusterRepo.UpdateProfile(ctx, clusterID, name, description)
}

// IssueClusterToken 为集群签发一枚新令牌；已有令牌时即为轮换，旧令牌随之失效。
//
// 入库的只有哈希与前缀；明文只经返回值交给调用方这一次，之后无处可取。
// ⛔ 明文与哈希都不写日志。集群未登记 → database.ErrClusterNotFound
func (s *ClusterRegistryService) IssueClusterToken(ctx context.Context, clusterID string) (*model.ClusterTokenIssued, error) {
	token, err := s.generateToken()
	if err != nil {
		return nil, fmt.Errorf("生成集群令牌: %w", err)
	}

	issuedAt := s.now()
	if err := s.clusterRepo.SetToken(ctx, clusterID, token.Hash, token.Prefix, issuedAt); err != nil {
		return nil, err
	}
	clusterRegistryLog.Info("已为集群签发令牌", "cluster", clusterID)

	return &model.ClusterTokenIssued{
		Token:         token.Plaintext,
		TokenPrefix:   token.Prefix,
		TokenIssuedAt: issuedAt.UTC().Format(time.RFC3339),
	}, nil
}

// UnregisterCluster 删除登记，令牌随之失效。
//
// 只删注册表里的这一行：该集群的历史数据（事件、指令历史、SLO 等）不受影响。
// 集群未登记 → database.ErrClusterNotFound
func (s *ClusterRegistryService) UnregisterCluster(ctx context.Context, clusterID string) error {
	if err := s.clusterRepo.Delete(ctx, clusterID); err != nil {
		return err
	}
	clusterRegistryLog.Info("集群登记已删除", "cluster", clusterID)
	return nil
}
