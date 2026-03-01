// atlhyper_master_v2/service/factory.go
// Service 工厂函数
package service

import (
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
)

// serviceImpl 组合 QueryService + CommandService
type serviceImpl struct {
	*query.QueryService
	*operations.CommandService
}

// NewService 创建统一 Service 实例
func NewService(q *query.QueryService, ops *operations.CommandService) Service {
	return &serviceImpl{QueryService: q, CommandService: ops}
}
