package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

var (
	apiCheckMu         sync.Mutex
	lastK8sStatus      = true
	K8sAvailable       = true
	checkInterval      = 15 * time.Second
	logThrottleSeconds = 30 * time.Second
)

var insecureHttpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func StartK8sHealthChecker(cfg *rest.Config) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		rt, err := rest.TransportFor(cfg)
		if err != nil {
			Error(context.TODO(), "❌ 无法构造认证 HTTP 客户端", zap.Error(err))
			os.Exit(1) // ❗初始化失败立即退出
		}

		client := &http.Client{Transport: rt}

		for range ticker.C {
			resp, err := client.Get(cfg.Host + "/healthz")

			apiCheckMu.Lock()
			healthy := err == nil && resp != nil && resp.StatusCode == 200
			K8sAvailable = healthy

			if !healthy {
				Error(context.TODO(), "🚨 无法连接 Kubernetes API Server", zap.Error(err))
				fmt.Println("❌ 无法访问 Kubernetes API Server，程序即将退出")
				os.Exit(1) // ❗一旦失联，立即退出程序
			} else if !lastK8sStatus {
				Info(context.TODO(), "✅ 成功重新连接 Kubernetes API Server")
			}
			lastK8sStatus = healthy
			apiCheckMu.Unlock()
		}
	}()
}
