// ======================================================
// 📄 文件：embedding/client/gemini_embedding_client.go
// 功能：Gemini Embedding 向量生成客户端 (text-embedding-004)
// ======================================================

package client

import (
	"AtlHyper/atlhyper_aiservice/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GeminiEmbeddingClient —— Gemini embedding 模型客户端
type GeminiEmbeddingClient struct {
	apiKey string
	model  string
}

// NewGeminiEmbeddingClient —— 工厂函数
func NewGeminiEmbeddingClient() *GeminiEmbeddingClient {
	cfg := config.GetGeminiConfig()
	model := cfg.EmbeddingModelName
	if model == "" {
		model = "text-embedding-004" // ✅ 冗余安全保障
	}

	return &GeminiEmbeddingClient{
		apiKey: cfg.APIKey,
		model:  model,
	}
}

// GenerateEmbedding —— 调用 Gemini embedding API 生成语义向量
func (c *GeminiEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// ✅ 新版 API：embedContent
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s",
		c.model, c.apiKey,
	)

	// ✅ 构造请求体
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": fmt.Sprintf("models/%s", c.model),
		"content": map[string]interface{}{
			"parts": []map[string]string{{"text": text}},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// ✅ 执行请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned %v", resp.Status)
	}

	// ✅ 解析响应
	var result struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result.Embedding.Values, nil
}

// Close —— 兼容接口，无需关闭资源
func (c *GeminiEmbeddingClient) Close() error { return nil }
