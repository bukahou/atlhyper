package config

import (
	"os"
	"time"
)

type MasterAPIConfig struct {
	BaseURL string
	Timeout time.Duration
}

const (
	defaultMasterURL   = "http://127.0.0.1:8081"
	envMasterURL       = "MASTER_API_URL"
	envMasterTimeout   = "MASTER_API_TIMEOUT"
)

func loadMasterConfig() MasterAPIConfig {
	var c MasterAPIConfig
	if v := os.Getenv(envMasterURL); v != "" {
		c.BaseURL = v
	} else {
		c.BaseURL = defaultMasterURL
	}

	if v := os.Getenv(envMasterTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		} else {
			c.Timeout = 8 * time.Second
		}
	} else {
		c.Timeout = 8 * time.Second
	}
	return c
}
