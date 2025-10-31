// atlhyper_aiservice/embedding/factory.go
package embedding

import (
	"AtlHyper/atlhyper_aiservice/embedding/client"
	"fmt"
	"strings"
)

// NewClientByFamily —— 根据家族标识创建客户端
func NewClientByFamily(family string) (EmbeddingClient, error) {
	switch {
	case strings.HasPrefix(family, "gemini"):
		return client.NewGeminiEmbeddingClient(), nil
	// case strings.HasPrefix(family, "gpt"):
	//     return client.NewOpenAIEmbeddingClient(), nil
	default:
		return nil, fmt.Errorf("unsupported embedding family: %s", family)
	}
}
