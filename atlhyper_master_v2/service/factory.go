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
	*operations.DeployService
	*operations.UserService
	*operations.ClusterRegistryService
}

// NewService 创建统一 Service 实例
func NewService(
	q *query.QueryService,
	cmd *operations.CommandService,
	admin *operations.AdminService,
	slo *operations.SLOService,
	deploy *operations.DeployService,
	user *operations.UserService,
	clusterRegistry *operations.ClusterRegistryService,
) Service {
	return &serviceImpl{
		QueryService:   q,
		CommandService: cmd,
		AdminService:   admin,
		SLOService:     slo,
		DeployService:  deploy,
		UserService:    user,

		ClusterRegistryService: clusterRegistry,
	}
}
