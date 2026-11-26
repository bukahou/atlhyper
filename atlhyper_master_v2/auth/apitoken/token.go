// Package apitoken 访问令牌：服务账号凭它调用对外入口（MCP）。
//
// 令牌 = atlp_ + 32 字节随机数；库里只存 SHA-256 与显示用前缀，明文只在签发时返回一次
// （生成、哈希与比对的做法与集群令牌相同，在 auth/opaquetoken）。
// 校验（Authenticator）每次都从库里读令牌与账号：吊销、到期、停用、改等级都立即生效。
package apitoken

import "AtlHyper/atlhyper_master_v2/auth/opaquetoken"

// plaintextPrefix 访问令牌的固定开头，便于在配置与日志里一眼认出这是什么
const plaintextPrefix = "atlp_"

// Issued 一枚刚生成的访问令牌
type Issued = opaquetoken.Issued

// Generate 生成一枚新访问令牌
func Generate() (Issued, error) {
	return opaquetoken.Generate(plaintextPrefix)
}

// Hash 计算访问令牌的入库哈希
func Hash(plaintext string) string {
	return opaquetoken.Hash(plaintext)
}
