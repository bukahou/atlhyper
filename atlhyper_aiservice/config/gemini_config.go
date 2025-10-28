package config

// GetGeminiConfig 返回 Gemini 子配置（供 client 使用）
func GetGeminiConfig() *GeminiConfig {
	return &C.Gemini
}

// GetServerConfig 返回 HTTP 服务配置
func GetServerConfig() *AIServerConfig {
	return &C.Server
}
