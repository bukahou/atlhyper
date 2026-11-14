// Package clustertoken 集群令牌的生成、哈希与校验
//
// 每个集群一个令牌，Agent 用它向 Master 证明自己属于哪个集群。
// 令牌由 Master 生成、只在签发时返回一次；库里只保存哈希。
package clustertoken

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// plaintextPrefix 令牌的固定开头，便于在配置与日志里一眼认出这是什么
	plaintextPrefix = "atlk_"
	randomBytes     = 32
	// displayPrefixLength 界面上用于辨认令牌的前缀长度（含固定开头）
	displayPrefixLength = 8
)

// Issued 一枚刚生成的令牌
type Issued struct {
	Plaintext string // 明文，只在签发时交给调用方一次
	Hash      string // 入库的哈希
	Prefix    string // 明文的前几个字符，供界面辨认
}

// Generate 生成一枚新令牌
func Generate() (Issued, error) {
	random := make([]byte, randomBytes)
	if _, err := rand.Read(random); err != nil {
		return Issued{}, fmt.Errorf("生成令牌随机数: %w", err)
	}
	plaintext := plaintextPrefix + base64.RawURLEncoding.EncodeToString(random)
	return Issued{
		Plaintext: plaintext,
		Hash:      Hash(plaintext),
		Prefix:    plaintext[:displayPrefixLength],
	}, nil
}

// Hash 计算令牌的入库哈希。
//
// 令牌是 256 位随机数，不存在可猜测的低熵输入，SHA-256 足够；
// 不需要 bcrypt 这类为口令设计的慢哈希，后者会拖慢每 10 秒一次的快照校验。
func Hash(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Matches 判断出示的令牌与库里的哈希是否相符（常数时间比较）。
// 任一侧为空都不相符：没签发过令牌的集群不能用空令牌通过。
func Matches(presented, storedHash string) bool {
	if presented == "" || storedHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(Hash(presented)), []byte(storedHash)) == 1
}

// FromBearerHeader 从 Authorization 头里取出令牌；不是 Bearer 形式时返回空串
func FromBearerHeader(header string) string {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
