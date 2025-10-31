// ======================================================
// 📦 文件：config/gemini_config.go
// 功能：加载 Gemini 模型与向量模型配置（生成 / Embedding）
// ======================================================

package config

import (
	"errors"
	"os"
)

// ========================================
// 🌐 Gemini 模型配置结构体
// ========================================
type GeminiConfig struct {
	APIKey             string // Google API 密钥
	ModelName          string // 文本生成模型名称（如 gemini-2.5-flash）
	EmbeddingModelName string // 向量模型名称（如 embedding-001）
}

// ========================================
// 🧩 默认值定义（Default Values）
// ========================================
const (
	defaultTextModelName      = "gemini-2.5-flash"
	defaultEmbeddingModelName = "text-embedding-004"
)

// ========================================
// 🌿 环境变量键名定义（Environment Keys）
// ========================================
const (
	envGeminiKey          = "GEMINI_API_KEY"           // API 密钥
	envGeminiTextModel    = "GEMINI_MODEL"             // 文本生成模型名
	envGeminiEmbeddingModel = "GEMINI_EMBEDDING_MODEL" // 向量模型名
)

// ========================================
// ⚙️ 加载逻辑
// ========================================
func loadGeminiConfig() (GeminiConfig, error) {
	var c GeminiConfig

	// ---------- API Key ----------
	key := os.Getenv(envGeminiKey)
	if key == "" {
		return c, errors.New("GEMINI_API_KEY 未设置")
	}
	c.APIKey = key

	// ---------- Text Model ----------
	if v := os.Getenv(envGeminiTextModel); v != "" {
		c.ModelName = v
	} else {
		c.ModelName = defaultTextModelName
	}

	// ---------- Embedding Model ----------
	if v := os.Getenv(envGeminiEmbeddingModel); v != "" {
		c.EmbeddingModelName = v
	} else {
		c.EmbeddingModelName = defaultEmbeddingModelName
	}

	return c, nil
}
