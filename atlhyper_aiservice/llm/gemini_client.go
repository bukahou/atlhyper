// atlhyper_aiservice/llm/gemini_client.go
package llm

import (
	"AtlHyper/atlhyper_aiservice/config"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// geminiClient —— 实现 LLMClient 接口
type geminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// newGeminiClient —— Gemini 模型客户端工厂
func newGeminiClient(ctx context.Context) (LLMClient, error) {
	cfg := config.GetGeminiConfig()
	c, err := genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, err
	}
	return &geminiClient{
		client: c,
		model:  c.GenerativeModel(cfg.ModelName),
	}, nil
}

func (g *geminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("generate text failed: %v", err)
	}
	var out strings.Builder
	for _, p := range resp.Candidates[0].Content.Parts {
		out.WriteString(fmt.Sprintf("%v", p))
	}
	return out.String(), nil
}

func (g *geminiClient) GenerateJSON(ctx context.Context, prompt string) (map[string]interface{}, error) {
	text, err := g.GenerateText(ctx, prompt)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		return map[string]interface{}{"raw": text}, nil
	}
	return m, nil
}

func (g *geminiClient) Close() error {
	return g.client.Close()
}



// // atlhyper_aiservice/client/ai/gemini_client.go
// package ai

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"sync"

// 	"AtlHyper/atlhyper_aiservice/config"

// 	"github.com/google/generative-ai-go/genai"
// 	"google.golang.org/api/option"
// )

// var (
// 	clientInstance *genai.Client
// 	once           sync.Once
// 	initErr        error
// )

// // InitGeminiClient 初始化 Gemini 客户端
// func InitGeminiClient() {
// 	_, err := GetGeminiClient(context.Background())
// 	if err != nil {
// 		log.Fatalf("❌ 初始化 Gemini 客户端失败: %v", err)
// 	}
// 	log.Println("✅ 成功初始化 Gemini 客户端")
// }

// // GetGeminiClient 单例模式
// func GetGeminiClient(ctx context.Context) (*genai.Client, error) {
// 	once.Do(func() {
// 		cfg := config.GetGeminiConfig()
// 		clientInstance, initErr = genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
// 		if initErr != nil {
// 			initErr = fmt.Errorf("创建 Gemini 客户端失败: %v", initErr)
// 			return
// 		}
// 	})
// 	return clientInstance, initErr
// }

// // CloseGeminiClient 关闭客户端
// func CloseGeminiClient() {
// 	if clientInstance != nil {
// 		clientInstance.Close()
// 	}
// }