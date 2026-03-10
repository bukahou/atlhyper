// atlhyper_master_v2/github/app.go
// GitHub App 身份管理 — JWT 签名 + Installation Token 自动刷新
package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"AtlHyper/common/logger"

	"github.com/golang-jwt/jwt/v5"
)

var log = logger.Module("GitHub")

// clientImpl GitHub 客户端实现
type clientImpl struct {
	cfg        Config
	privateKey *rsa.PrivateKey
	httpClient *http.Client

	// Installation Token 缓存
	mu              sync.RWMutex
	installationID  int64
	installToken    string
	installTokenExp time.Time
}

// NewClient 创建 GitHub 客户端
func NewClient(cfg Config) (Client, error) {
	c := &clientImpl{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// 加载 Private Key（可选，未配置则部分功能不可用）
	if cfg.PrivateKeyPath != "" {
		key, err := loadPrivateKey(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("加载 GitHub Private Key 失败: %w", err)
		}
		c.privateKey = key
		log.Info("GitHub App Private Key 加载成功")
	}

	return c, nil
}

// SetInstallationID 设置 Installation ID（从数据库恢复）
func (c *clientImpl) SetInstallationID(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.installationID = id
}

// signJWT 生成 GitHub App JWT（有效期 10 分钟）
func (c *clientImpl) signJWT() (string, error) {
	if c.privateKey == nil {
		return "", fmt.Errorf("private key not loaded")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iat": jwt.NewNumericDate(now.Add(-60 * time.Second)), // 允许时钟偏差
		"exp": jwt.NewNumericDate(now.Add(10 * time.Minute)),
		"iss": c.cfg.AppID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(c.privateKey)
}

// GetInstallationToken 获取 Installation Token（自动缓存刷新）
func (c *clientImpl) GetInstallationToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.installToken != "" && time.Now().Before(c.installTokenExp.Add(-5*time.Minute)) {
		token := c.installToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	return c.refreshInstallationToken(ctx)
}

// refreshInstallationToken 刷新 Installation Token
func (c *clientImpl) refreshInstallationToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查
	if c.installToken != "" && time.Now().Before(c.installTokenExp.Add(-5*time.Minute)) {
		return c.installToken, nil
	}

	if c.installationID == 0 {
		return "", fmt.Errorf("no installation ID configured")
	}

	jwtToken, err := c.signJWT()
	if err != nil {
		return "", fmt.Errorf("JWT 签名失败: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", c.installationID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 Installation Token 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("获取 Installation Token 失败: %d %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	c.installToken = result.Token
	c.installTokenExp = result.ExpiresAt

	log.Info("Installation Token 已刷新", "expiresAt", result.ExpiresAt.Format(time.RFC3339))
	return c.installToken, nil
}

// loadPrivateKey 从 PEM 文件加载 RSA 私钥
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// 尝试 PKCS8 格式
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("PKCS1: %w, PKCS8: %w", err, err2)
		}
		rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		return rsaKey, nil
	}

	return key, nil
}
