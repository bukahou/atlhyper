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

package utils

// import (
// 	"crypto/tls"
// 	"log"
// 	"net/http"
// 	"os"
// 	"sync"

// 	"k8s.io/client-go/rest"
// )

// var (
// 	apiCheckMu    sync.Mutex
// 	lastK8sStatus = true // 上一次检测的 Kubernetes API 状态
// 	K8sAvailable  = true // 当前 Kubernetes API 是否可用
// 	//外部定义
// 	// checkInterval = 15 * time.Second
// 	// 日志打印节流时间（预留未使用）
// 	// logThrottleSeconds = 30 * time.Second
// )

// // 不安全的 HTTP 客户端（用于跳过 TLS 验证，仅供内部使用）
// var insecureHttpClient = &http.Client{
// 	Transport: &http.Transport{
// 		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
// 	},
// }

// // ✅ 启动后台协程持续监测 Kubernetes API Server 健康状态
// // StartK8sHealthChecker ✅ 执行一次 Kubernetes API Server 健康探测
// func StartK8sHealthChecker(cfg *rest.Config) {
// 	rt, err := rest.TransportFor(cfg)
// 	if err != nil {
// 		log.Printf("❌ 初始化 Kubernetes Transport 失败: %v", err)
// 		os.Exit(1)
// 	}

// 	client := &http.Client{Transport: rt}
// 	resp, err := client.Get(cfg.Host + "/healthz")

// 	apiCheckMu.Lock()
// 	defer apiCheckMu.Unlock()

// 	healthy := err == nil && resp != nil && resp.StatusCode == 200
// 	K8sAvailable = healthy

// 	if !healthy {
// 		log.Println("❌ 无法访问 Kubernetes API Server，即将退出...")
// 		os.Exit(1)
// 	}

// 	// 状态发生变化时可扩展日志或告警
// 	lastK8sStatus = healthy
// }
