package utils

import (
	"log"
	"sync"

	"AtlHyper/atlhyper_agent/config"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient     client.Client         // controller-runtime
	coreClientset *kubernetes.Clientset // client-go
	once          sync.Once
	cfg           *rest.Config // 存储解析得到的 Kubernetes 配置
)

// InitK8sClient 初始化 Kubernetes 客户端配置（rest.Config）
// 支持从配置加载 kubeconfig 路径，也支持 InCluster 模式
func InitK8sClient() *rest.Config {
	once.Do(func() {
		var err error

		// 从配置获取 kubeconfig 路径
		kubeconfig := config.GlobalConfig.Kubernetes.Kubeconfig
		if kubeconfig != "" {
			// 若环境变量存在，尝试使用该路径构建配置
			cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				log.Printf("⚠️ 使用 KUBECONFIG 构建失败: %v", err)
			} else {
				log.Printf("✅ 成功加载 kubeconfig: %s", kubeconfig)
			}
		}

		// 如果 cfg 仍然为 nil，说明 kubeconfig 加载失败，尝试 InCluster 模式（用于 Pod 内运行）
		if cfg == nil {
			cfg, err = rest.InClusterConfig()
			if err != nil {
				log.Printf("获取 in-cluster 配置失败: %v", err)
				panic(err) // 无法继续运行，直接终止程序
			}
		}

		// 使用构建好的配置初始化 controller-runtime 的 k8s client
		k8sClient, err = client.New(cfg, client.Options{})
		if err != nil {
			log.Printf("初始化 k8sClient 失败: %v", err)
			panic(err) // 客户端初始化失败也不能继续运行
		}

		coreClientset, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			log.Fatalf("初始化 client-go client 失败: %v", err)
		}

		log.Println("✅ 成功初始化 controller-runtime 与 client-go 客户端")
	})

	// 返回初始化好的配置
	return cfg
}

// 获取全局共享的 controller-runtime client 实例
func GetClient() client.Client {
	if k8sClient == nil {
		panic("k8sClient 为 nil")
	}
	return k8sClient
}

// 返回共享的 rest.Config，若未初始化则 panic
func GetRestConfig() *rest.Config {
	if cfg == nil {
		panic("rest.Config 未初始化，请先调用 InitK8sClient()")
	}
	return cfg
}

// 获取全局共享的 client-go client 实例（CoreV1、AppsV1 等）
func GetCoreClient() *kubernetes.Clientset {
	if coreClientset == nil {
		panic("client-go CoreClient 未初始化，请先调用 InitK8sClient()")
	}
	return coreClientset
}
