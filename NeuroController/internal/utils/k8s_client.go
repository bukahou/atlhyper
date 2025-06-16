// =======================================================================================
// 📄 k8s_client.go
//
// ✨ Description:
//     Encapsulates controller-runtime's Kubernetes client initialization logic,
//     providing a globally shared client.Client instance for modules such as Watcher,
//     Scaler, Webhook, etc.
//
// 🛠️ Provided Functions:
//     - InitK8sClient(): Initializes the client.Client (thread-safe, runs once)
//     - GetClient(): Returns the initialized global client.Client instance
//
// 📦 Dependencies:
//     - sigs.k8s.io/controller-runtime/pkg/client
//     - sigs.k8s.io/controller-runtime/pkg/client/config
//
// 📍 Usage:
//     - Call InitK8sClient() once at controller startup
//     - Other modules retrieve the shared client via utils.GetClient()
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package utils

import (
	"os"
	"sync"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient client.Client
	once      sync.Once
	cfg       *rest.Config // 存储解析得到的 Kubernetes 配置
)

// 初始化全局的 controller-runtime client.Client 实例
func InitK8sClient() *rest.Config {
	once.Do(func() {
		var err error

		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig != "" {
			cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err == nil {
			} else {
			}
		}

		if cfg == nil {
			cfg, err = rest.InClusterConfig()
			if err != nil {
				panic(err)
			}
		}

		k8sClient, err = client.New(cfg, client.Options{})
		if err != nil {
			panic(err)
		}

	})
	return cfg
}

// 获取全局共享的 controller-runtime client 实例
func GetClient() client.Client {
	if k8sClient == nil {
		panic("k8sClient 为 nil")
	}
	return k8sClient
}
