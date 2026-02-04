// common/crypto/aes.go
// AES-256-GCM 加解密工具
// 用于敏感数据（如 API Key）的加密存储
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

// 加密前缀，用于识别已加密的数据
const encryptedPrefix = "enc:"

// Encryptor 加密器
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor 创建加密器
// secret: 任意长度的密钥（会通过 SHA-256 转换为 32 字节）
func NewEncryptor(secret string) (*Encryptor, error) {
	if secret == "" {
		return nil, errors.New("encryption secret cannot be empty")
	}

	// 使用 SHA-256 将任意长度密钥转换为 32 字节（AES-256）
	hash := sha256.Sum256([]byte(secret))
	key := hash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{gcm: gcm}, nil
}

// Encrypt 加密明文
// 返回 "enc:" 前缀 + base64 编码的密文
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 如果已经加密过，直接返回
	if strings.HasPrefix(plaintext, encryptedPrefix) {
		return plaintext, nil
	}

	// 生成随机 nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密
	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回带前缀的 base64 编码
	return encryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密密文
// 自动处理带 "enc:" 前缀的密文，无前缀视为明文直接返回
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// 如果没有加密前缀，视为明文返回（兼容旧数据）
	if !strings.HasPrefix(ciphertext, encryptedPrefix) {
		return ciphertext, nil
	}

	// 去掉前缀并 base64 解码
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, encryptedPrefix))
	if err != nil {
		return "", err
	}

	// 提取 nonce
	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := e.gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// MaskAPIKey 遮蔽 API Key 用于显示
// 例: "AIzaSyXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX" -> "AIza****XXXX"
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	// 已加密的不显示
	if strings.HasPrefix(apiKey, encryptedPrefix) {
		return "****"
	}

	length := len(apiKey)
	if length <= 8 {
		return "****"
	}

	// 显示前4位和后4位
	return apiKey[:4] + "****" + apiKey[length-4:]
}
