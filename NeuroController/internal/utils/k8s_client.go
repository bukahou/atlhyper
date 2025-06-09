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
	cfg       *rest.Config // Stores the resolved config
)

// InitK8sClient initializes the global controller-runtime client.Client instance
func InitK8sClient() *rest.Config {
	once.Do(func() {
		var err error

		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig != "" {
			cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err == nil {
				Info(context.TODO(), "✅ Initialized using local kubeconfig")
			} else {
				Warn(context.TODO(), "⚠️ Failed to parse local kubeconfig, falling back to in-cluster", zap.Error(err))
			}
		}

		if cfg == nil {
			cfg, err = rest.InClusterConfig()
			if err != nil {
				Error(context.TODO(), "❌ Failed to load in-cluster Kubernetes configuration", zap.Error(err))
				panic(err)
			}
			Info(context.TODO(), "✅ Initialized using in-cluster configuration")
		}

		k8sClient, err = client.New(cfg, client.Options{})
		if err != nil {
			Error(context.TODO(), "❌ Failed to initialize Kubernetes client", zap.Error(err))
			panic(err)
		}

		Info(context.TODO(), "✅ Kubernetes client successfully initialized")
	})
	return cfg
}

// GetClient returns the globally shared controller-runtime client
func GetClient() client.Client {
	if k8sClient == nil {
		Error(context.TODO(), "⛔ GetClient() called before InitK8sClient()")
		panic("k8sClient is nil")
	}
	return k8sClient
}
