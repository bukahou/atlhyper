// atlhyper_master_v2/service/apitoken_contract.go
// 编译期核对：统一 Service 满足访问令牌校验器要的两个最小接口，入口层只拿 Service 就能构造校验器，
// 不直接碰数据库。放在 service 包里：apitoken 不引用 service，不会成环
package service

import "AtlHyper/atlhyper_master_v2/auth/apitoken"

var (
	_ apitoken.TokenStore = (Service)(nil)
	_ apitoken.UserLoader = (Service)(nil)
)
