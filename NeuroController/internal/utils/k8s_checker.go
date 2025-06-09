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

// =======================================================================================
// 📄 k8s_checker.go
//
// ✨ Description:
//     Periodically checks the health status of the Kubernetes API server via /healthz.
//     Terminates the program if API is unreachable, to prevent operating in a degraded state.
//
// 📦 Behavior:
//     - Runs on a fixed interval (default 15s)
//     - Logs success or failure with throttling
//     - Sets global `K8sAvailable` status flag
// =======================================================================================

var (
	apiCheckMu         sync.Mutex
	lastK8sStatus      = true
	K8sAvailable       = true
	checkInterval      = 15 * time.Second
	logThrottleSeconds = 30 * time.Second
)

// Insecure client for internal use (e.g., bypassing TLS validation)
var insecureHttpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// ✅ Starts a background goroutine to monitor Kubernetes API health
func StartK8sHealthChecker(cfg *rest.Config) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		rt, err := rest.TransportFor(cfg)
		if err != nil {
			Error(context.TODO(), "❌ Failed to build authenticated HTTP client", zap.Error(err))
			os.Exit(1) // ❗ Exit immediately if initialization fails
		}

		client := &http.Client{Transport: rt}

		for range ticker.C {
			resp, err := client.Get(cfg.Host + "/healthz")

			apiCheckMu.Lock()
			healthy := err == nil && resp != nil && resp.StatusCode == 200
			K8sAvailable = healthy

			if !healthy {
				Error(context.TODO(), "Unable to connect to Kubernetes API Server", zap.Error(err))
				fmt.Println("❌ Kubernetes API Server is unreachable, terminating...")
				os.Exit(1) // ❗ Exit immediately on disconnection
			} else if !lastK8sStatus {
				Info(context.TODO(), "✅ Successfully reconnected to Kubernetes API Server")
			}

			lastK8sStatus = healthy
			apiCheckMu.Unlock()
		}
	}()
}
