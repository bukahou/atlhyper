package config

import (
	"errors"
	"os"
	"time"
)

type GeminiConfig struct {
	APIKey    string
	ModelName string
	Timeout   time.Duration
}

const (
	defaultModelName   = "gemini-2.5-flash"
	envGeminiKey       = "GEMINI_API_KEY"
	envGeminiModel     = "GEMINI_MODEL"
	envGeminiTimeout   = "GEMINI_TIMEOUT"
)

func loadGeminiConfig() (GeminiConfig, error) {
	var c GeminiConfig

	key := os.Getenv(envGeminiKey)
	if key == "" {
		return c, errors.New("GEMINI_API_KEY 未设置")
	}
	c.APIKey = key

	if v := os.Getenv(envGeminiModel); v != "" {
		c.ModelName = v
	} else {
		c.ModelName = defaultModelName
	}

	if val := os.Getenv(envGeminiTimeout); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			c.Timeout = d
		} else {
			c.Timeout = 10 * time.Second
		}
	} else {
		c.Timeout = 10 * time.Second
	}

	return c, nil
}
