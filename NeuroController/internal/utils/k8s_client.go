// =======================================================================================
// 📄 k8s_client.go
//
// ✨ Description:
//     Encapsulates the initialization of both controller-runtime and client-go Kubernetes clients,
//     providing globally shared instances to be used across modules such as Watcher, Diagnosis,
//     Webhook, etc.
//
// 🛠️ Provided Functions:
//     - InitK8sClient(): Initializes Kubernetes clients (controller-runtime + client-go)
//     - GetClient(): Returns the shared controller-runtime client
//     - GetRestConfig(): Returns the loaded rest.Config
//     - GetCoreClient(): Returns the shared client-go CoreV1 clientset
//
// 📦 Features:
//     - Supports KUBECONFIG-based or InCluster configuration
//     - Thread-safe initialization using sync.Once
//     - Panics on critical initialization failure
//
// 📍 Usage:
//     - Call InitK8sClient() during startup
//     - Use GetClient() / GetCoreClient() in other modules
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package utils

import (
	"log"
	"os"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient     client.Client         // controller-runtime
	coreClientset *kubernetes.Clientset // client-go
	// k8sClient     client.Client
	once sync.Once
	cfg  *rest.Config // 存储解析得到的 Kubernetes 配置
)

// 初始化全局的 controller-runtime client.Client 实例
// InitK8sClient 初始化 Kubernetes 客户端配置（rest.Config）
// 支持从 KUBECONFIG 环境变量加载配置，也支持 InCluster 模式
func InitK8sClient() *rest.Config {
	// once.Do 确保只执行一次初始化（线程安全）
	once.Do(func() {
		var err error

		// 尝试从环境变量 KUBECONFIG 读取 kubeconfig 路径
		kubeconfig := os.Getenv("KUBECONFIG")
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
