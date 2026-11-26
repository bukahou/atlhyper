// atlhyper_master_v2/service/query_contract.go
// 编译期核对：每个查询功能域小服务满足对应的子接口，组合后的 *QueryService 满足 Query。
// 放在 service 包里，因为 query 包不能反向引用 service（会成环）
package service

import "AtlHyper/atlhyper_master_v2/service/query"

var (
	_ QueryK8s             = (*query.K8sQuery)(nil)
	_ QueryOTel            = (*query.OTelQuery)(nil)
	_ QuerySLO             = (*query.SLOQuery)(nil)
	_ QueryAIOps           = (*query.AIOpsQuery)(nil)
	_ QueryOverview        = (*query.OverviewQuery)(nil)
	_ QueryAdmin           = (*query.AdminQuery)(nil)
	_ QueryGitHub          = (*query.GitHubQuery)(nil)
	_ QueryUser            = (*query.UserQuery)(nil)
	_ QueryClusterRegistry = (*query.ClusterRegistryQuery)(nil)
	_ QueryNotify          = (*query.NotifyQuery)(nil)
	_ QueryServiceAccounts = (*query.ServiceAccountQuery)(nil)
	_ Query                = (*query.QueryService)(nil)
)
