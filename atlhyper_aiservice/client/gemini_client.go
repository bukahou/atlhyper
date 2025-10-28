// atlhyper_aiservice/client/gemini_client.go
package client

import (
	"context"
	"fmt"
	"log"
	"sync"

	"AtlHyper/atlhyper_aiservice/config"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var (
	clientInstance *genai.Client
	once           sync.Once
	initErr        error
)

// InitGeminiClient 初始化 Gemini 客户端
func InitGeminiClient() {
	_, err := GetGeminiClient(context.Background())
	if err != nil {
		log.Fatalf("❌ 初始化 Gemini 客户端失败: %v", err)
	}
	log.Println("✅ 成功初始化 Gemini 客户端")
}

// GetGeminiClient 单例模式
func GetGeminiClient(ctx context.Context) (*genai.Client, error) {
	once.Do(func() {
		cfg := config.GetGeminiConfig()
		clientInstance, initErr = genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
		if initErr != nil {
			initErr = fmt.Errorf("创建 Gemini 客户端失败: %v", initErr)
			return
		}
	})
	return clientInstance, initErr
}

// CloseGeminiClient 关闭客户端
func CloseGeminiClient() {
	if clientInstance != nil {
		clientInstance.Close()
	}
}

