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
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
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
				Info(context.TODO(), "✅ 使用本地 kubeconfig 初始化成功")
			} else {
				Warn(context.TODO(), "⚠️ 解析本地 kubeconfig 失败，回退为集群内配置", zap.Error(err))
			}
		}

		if cfg == nil {
			cfg, err = rest.InClusterConfig()
			if err != nil {
				Error(context.TODO(), "❌ 加载集群内 Kubernetes 配置失败", zap.Error(err))
				panic(err)
			}
			Info(context.TODO(), "✅ 使用集群内配置初始化成功")
		}

		k8sClient, err = client.New(cfg, client.Options{})
		if err != nil {
			Error(context.TODO(), "❌ 初始化 Kubernetes 客户端失败", zap.Error(err))
			panic(err)
		}

		Info(context.TODO(), "✅ Kubernetes 客户端初始化完成")
	})
	return cfg
}

// 获取全局共享的 controller-runtime client 实例
func GetClient() client.Client {
	if k8sClient == nil {
		Error(context.TODO(), "⛔ 在调用 InitK8sClient() 之前调用了 GetClient()")
		panic("k8sClient 为 nil")
	}
	return k8sClient
}
