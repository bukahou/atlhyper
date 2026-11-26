// Package opaquetoken 不透明令牌的生成、哈希与比对：集群令牌与访问令牌共用的一套做法。
//
// 令牌 = 固定开头 + 32 字节随机数（base64url，无填充）。库里只存 SHA-256（十六进制）与显示用前缀，
// 明文只在签发时交给调用方一次。各种令牌只差固定开头，由使用方的包各自定义。
package opaquetoken

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
	randomBytes = 32
	// displayRandomChars 显示用前缀里随机部分的字符数：够在界面上区分同一账号的几枚令牌，
	// 又只透露 18 位随机数（余下 238 位）
	displayRandomChars = 3
)

// Issued 一枚刚生成的令牌
type Issued struct {
	Plaintext string // 明文，只在签发时交给调用方一次
	Hash      string // 入库的哈希
	Prefix    string // 固定开头 + 随机部分的前几个字符，供界面辨认
}

// Generate 生成一枚以 prefix 开头的新令牌
func Generate(prefix string) (Issued, error) {
	random := make([]byte, randomBytes)
	if _, err := rand.Read(random); err != nil {
		return Issued{}, fmt.Errorf("生成令牌随机数: %w", err)
	}
	plaintext := prefix + base64.RawURLEncoding.EncodeToString(random)
	return Issued{
		Plaintext: plaintext,
		Hash:      Hash(plaintext),
		Prefix:    plaintext[:len(prefix)+displayRandomChars],
	}, nil
}

// Hash 计算令牌的入库哈希。
//
// 令牌是 256 位随机数，不存在可猜测的低熵输入，SHA-256 足够；
// 不需要 bcrypt 这类为口令设计的慢哈希，后者会拖慢每个请求的校验。
func Hash(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Matches 判断出示的令牌与库里的哈希是否相符（常数时间比较）。
// 任一侧为空都不相符：没签发过令牌的记录不能用空令牌通过。
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
