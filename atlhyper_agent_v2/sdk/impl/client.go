// Package impl K8sClient 接口的具体实现
//
// client.go - 客户端结构体定义和初始化
//
// 本文件定义 k8sClient 结构体，并提供初始化函数。
// 具体的 API 操作分布在其他文件中：
//   - core.go: corev1 资源 (Pod, Node, Service, ConfigMap, Secret, Namespace, Event, PV, PVC)
//   - apps.go: appsv1 资源 (Deployment, StatefulSet, DaemonSet, ReplicaSet)
//   - batch.go: batchv1 资源 (Job, CronJob)
//   - networking.go: networkingv1 资源 (Ingress)
//   - metrics.go: metrics 资源 (NodeMetrics)
//   - generic.go: 通用操作 (Delete, Dynamic)
package impl

import (
	"fmt"
	"log"

	"AtlHyper/atlhyper_agent_v2/sdk"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Client K8s 客户端实现
//
// 实现 sdk.K8sClient 接口。
// 包含:
//   - clientset: client-go 的 Kubernetes 客户端
//   - metricsClient: metrics-server 客户端 (可选)
//   - config: REST 配置 (用于 exec/logs 等流式操作)
type Client struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset // 可能为 nil (集群未安装 metrics-server)
	config        *rest.Config
}

// NewClient 创建 K8s 客户端实现
//
// kubeconfig 参数:
//   - 空字符串: 使用 in-cluster 配置 (Pod 内运行时)
//   - 文件路径: 使用指定的 kubeconfig 文件 (本地调试时)
func NewClient(kubeconfig string) (sdk.K8sClient, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build k8s config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	// 初始化 metrics client (可选，失败不影响其他功能)
	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		log.Printf("[SDK] metrics 客户端初始化失败（非致命）: %v", err)
		metricsClient = nil
	}

	return &Client{
		clientset:     clientset,
		metricsClient: metricsClient,
		config:        config,
	}, nil
}
