// atlhyper_master_v2/service/factory.go
// Service 工厂函数
package service

import (
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
)

// serviceImpl 组合 QueryService 与各写入服务
type serviceImpl struct {
	*query.QueryService
	*operations.CommandService
	*operations.AdminService
	*operations.SLOService
	*operations.GitHubService
	*operations.UserService
	*operations.ClusterRegistryService
	*operations.NotifyService
	*operations.ServiceAccountService
}

// NewService 创建统一 Service 实例
func NewService(
	q *query.QueryService,
	cmd *operations.CommandService,
	admin *operations.AdminService,
	slo *operations.SLOService,
	github *operations.GitHubService,
	user *operations.UserService,
	clusterRegistry *operations.ClusterRegistryService,
	notify *operations.NotifyService,
	serviceAccounts *operations.ServiceAccountService,
) Service {
	return &serviceImpl{
		QueryService:   q,
		CommandService: cmd,
		AdminService:   admin,
		SLOService:     slo,
		GitHubService:  github,
		UserService:    user,

		ClusterRegistryService: clusterRegistry,
		NotifyService:          notify,
		ServiceAccountService:  serviceAccounts,
	}
}
