// Package clustertoken 集群令牌的生成、哈希与校验
//
// 每个集群一个令牌，Agent 用它向 Master 证明自己属于哪个集群。
// 令牌由 Master 生成、只在签发时返回一次；库里只保存哈希。
// 生成、哈希与比对的做法与访问令牌相同，在 auth/opaquetoken；这里只定固定开头。
package clustertoken

import "AtlHyper/atlhyper_master_v2/auth/opaquetoken"

// plaintextPrefix 令牌的固定开头，便于在配置与日志里一眼认出这是什么
const plaintextPrefix = "atlk_"

// Issued 一枚刚生成的令牌
type Issued = opaquetoken.Issued

// Generate 生成一枚新令牌
func Generate() (Issued, error) {
	return opaquetoken.Generate(plaintextPrefix)
}

// Hash 计算令牌的入库哈希
func Hash(plaintext string) string {
	return opaquetoken.Hash(plaintext)
}

// Matches 判断出示的令牌与库里的哈希是否相符（常数时间比较）。
// 任一侧为空都不相符：没签发过令牌的集群不能用空令牌通过。
func Matches(presented, storedHash string) bool {
	return opaquetoken.Matches(presented, storedHash)
}

// FromBearerHeader 从 Authorization 头里取出令牌；不是 Bearer 形式时返回空串
func FromBearerHeader(header string) string {
	return opaquetoken.FromBearerHeader(header)
}
