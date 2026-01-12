package bootstrap

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent/config"
	"AtlHyper/atlhyper_agent/source/event/datahub"
	"AtlHyper/atlhyper_agent/sdk"

	"k8s.io/client-go/rest"
)

// StartCleanSystem 启动清理器协程，用于定期清理原始事件并存储至清理池。
// 该任务通过 config 中的 CleanInterval 控制清理周期。
func StartCleanSystem() {
	// 读取清理周期配置
	interval := config.GlobalConfig.Diagnosis.CleanInterval

	// 打印启动日志（带周期信息）
	log.Printf("✅ [Startup] 清理器启动（周期: %s）", interval)

	// 启动一个后台协程，定期调用事件清理逻辑
	go func() {
		for {
			// 调用清理函数：去重、聚合、生成告警候选
			datahub.CleanAndStoreEvents()

			// 等待下一周期
			time.Sleep(interval)
		}
	}()
}

// Startclientchecker 启动 Kubernetes 集群健康检查器。
// 内部通过 API Server /healthz 探针检测集群是否可用。
func Startclientchecker() {
	log.Println("✅ [Startup] 启动集群健康检查器")

	cfg := sdk.Get().RestConfig() // 💡 通过 SDK 获取配置
	interval := config.GlobalConfig.Kubernetes.APIHealthCheckInterval

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// 立即执行一次
		StartK8sHealthChecker(cfg)

		for range ticker.C {
			StartK8sHealthChecker(cfg)
		}
	}()
}


var (
	apiCheckMu    sync.Mutex
	lastK8sStatus = true // 上一次检测的 Kubernetes API 状态
	K8sAvailable  = true // 当前 Kubernetes API 是否可用
	//外部定义
	// checkInterval = 15 * time.Second
	// 日志打印节流时间（预留未使用）
	// logThrottleSeconds = 30 * time.Second
)

// 不安全的 HTTP 客户端（用于跳过 TLS 验证，仅供内部使用）
var insecureHttpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// ✅ 启动后台协程持续监测 Kubernetes API Server 健康状态
// StartK8sHealthChecker ✅ 执行一次 Kubernetes API Server 健康探测
func StartK8sHealthChecker(cfg *rest.Config) {
	rt, err := rest.TransportFor(cfg)
	if err != nil {
		log.Printf("❌ 初始化 Kubernetes Transport 失败: %v", err)
		os.Exit(1)
	}

	client := &http.Client{Transport: rt}
	resp, err := client.Get(cfg.Host + "/healthz")

	apiCheckMu.Lock()
	defer apiCheckMu.Unlock()

	healthy := err == nil && resp != nil && resp.StatusCode == 200
	K8sAvailable = healthy

	if !healthy {
		log.Println("❌ 无法访问 Kubernetes API Server，即将退出...")
		os.Exit(1)
	}

	// 状态发生变化时可扩展日志或告警
	lastK8sStatus = healthy
}