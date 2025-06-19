// =======================================================================================
// 📄 internal/bootstrap/manager.go
//
// ✨ Description:
//     Encapsulates the startup logic for controller-runtime's Manager,
//     responsible for initializing and starting all registered Watchers.
//     Serves as the primary bootstrap module invoked from cmd/neurocontroller/main.go,
//     decoupling main function from core registration and lifecycle logic.
//
// 📦 Features:
//     - StartManager(): Initializes and starts the controller-runtime manager.
//     - Handles kubeconfig resolution (external or in-cluster)
//     - Registers and runs all Watcher modules via watcher.RegisterAllWatchers
//
// 📍 Usage:
//     - Call StartManager() from main.go to launch controller lifecycle.
//
// 🧩 Dependencies:
//     - sigs.k8s.io/controller-runtime
//     - watcher.RegisterAllWatchers
//     - client-go rest/config resolution
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package bootstrap

import (
	"NeuroController/internal/watcher"
	"context"
	"log"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ✅ 启动控制器管理器（加载并运行所有 Watcher 模块）
func StartManager() {
	// ✅ 创建 controller-runtime 的管理器
	cfg, err := resolveRestConfig()
	if err != nil {
		log.Printf("❌ 无法解析 Kubernetes 配置: %v", err)
		return
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{})
	if err != nil {
		log.Printf("❌ 无法创建 controller manager: %v", err)
		return
	}

	// ✅ 注册所有 Watcher 模块
	if err := watcher.RegisterAllWatchers(mgr); err != nil {
		log.Printf("❌ Watcher 模块注册失败: %v", err)
		return
	}

	// ✅ 启动控制器主循环（阻塞调用）
	if err := mgr.Start(context.Background()); err != nil {
		log.Printf("❌ 控制器主循环启动失败: %v", err)
		return
	}
}

// ✅ 私有辅助函数：自动检测 kubeconfig 或集群内配置
func resolveRestConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err == nil {
			return cfg, nil
		}
		log.Printf("⚠️ 使用 kubeconfig 加载失败，将尝试使用 InClusterConfig: %v", err)
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
