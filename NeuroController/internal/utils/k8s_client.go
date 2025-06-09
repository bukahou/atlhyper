// =======================================================================================
// 📄 k8s_client.go
//
// ✨ 功能说明：
//     本模块封装了 controller-runtime 的 Kubernetes 客户端初始化逻辑，
//     统一提供 client.Client 实例供 Watcher、Scaler、Webhook 等模块共享访问。
//     支持自动判断 InCluster 与本地 kubeconfig，适配开发与集群环境。
//
// 🛠️ 提供功能：
//     - InitK8sClient(): 初始化 client.Client（线程安全，仅执行一次）
//     - GetClient(): 获取已初始化的 client.Client 实例
//
// 📦 依赖：
//     - controller-runtime (sigs.k8s.io/controller-runtime/pkg/client)
//     - controller-runtime 配置管理 (sigs.k8s.io/controller-runtime/pkg/client/config)
//
// 📍 使用方式：
//     - 在 controller 启动时先调用 InitK8sClient()
//     - 后续模块通过 utils.GetClient() 获取共享 client 实例
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
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
	cfg       *rest.Config //  保存 config
)

// InitK8sClient 初始化 controller-runtime 的 Client
func InitK8sClient() *rest.Config {
	once.Do(func() {
		// var cfg *rest.Config
		var err error

		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig != "" {
			cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err == nil {
				Info(context.TODO(), "✅ 使用本地 kubeconfig 初始化")
			} else {
				Warn(context.TODO(), "⚠️ 解析本地 kubeconfig 失败，尝试 InCluster", zap.Error(err))
			}
		}

		if cfg == nil {
			cfg, err = rest.InClusterConfig()
			if err != nil {
				Error(context.TODO(), "❌ 无法加载 Kubernetes 配置", zap.Error(err))
				panic(err)
			}
			Info(context.TODO(), "✅ 使用集群内配置初始化")
		}

		k8sClient, err = client.New(cfg, client.Options{})
		if err != nil {
			Error(context.TODO(), "❌ 无法初始化 Kubernetes 客户端", zap.Error(err))
			panic(err)
		}

		Info(context.TODO(), "✅ Kubernetes 客户端初始化完成")
	})
	return cfg
}

// GetClient 返回全局共享的 controller-runtime Client
func GetClient() client.Client {
	if k8sClient == nil {
		Error(context.TODO(), "⛔ GetClient() 调用前未初始化 k8s client")
		panic("k8sClient is nil")
	}
	return k8sClient
}
