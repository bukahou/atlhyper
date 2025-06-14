// =======================================================================================
// 📄 internal/bootstrap/manager.go
//
// ✨ Description:
//     Encapsulates the startup logic of controller-runtime's manager,
//     responsible for loading all Watchers and starting the control loop.
//     Acts as the core bootstrap module for cmd/neurocontroller/main.go,
//     decoupling the main function from registration logic.
//
// 📦 Provided Features:
//     - StartManager(): Starts the controller-runtime manager.
//
// 📍 Usage Scenario:
//     - Called by main.go as the unified entry point to launch controllers.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package bootstrap

import (
	"NeuroController/internal/utils"
	"NeuroController/internal/watcher"
	"context"
	"os"

	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ✅ 启动控制器管理器（加载并运行所有 Watcher 模块）
func StartManager() {
	// ✅ 创建 controller-runtime 的管理器
	cfg, err := resolveRestConfig()
	if err != nil {
		utils.Fatal(nil, "❌ 加载 Kubernetes 配置失败", zap.Error(err))
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		// 为未来支持命名空间过滤预留。目前监控整个集群。
		//Namespace: "default",
	})
	if err != nil {
		utils.Fatal(nil, "❌ 初始化控制器管理器失败", zap.Error(err))
	}

	// ✅ 注册所有 Watcher 模块
	if err := watcher.RegisterAllWatchers(mgr); err != nil {
		utils.Fatal(nil, "❌ 注册 Watcher 模块失败", zap.Error(err))
	}

	// ✅ 启动控制器主循环（阻塞调用）
	utils.Info(nil, "🚀 正在启动 controller-runtime 管理器 ...")
	if err := mgr.Start(context.Background()); err != nil {
		utils.Fatal(nil, "❌ 控制器主循环异常退出", zap.Error(err))
	}
}

// ✅ 私有辅助函数：自动检测 kubeconfig 或集群内配置
func resolveRestConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err == nil {
			utils.Info(context.TODO(), "✅ 使用本地 kubeconfig 配置")
			return cfg, nil
		}
		utils.Warn(context.TODO(), "⚠️ 读取本地 kubeconfig 失败，尝试使用集群内配置", zap.Error(err))
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		utils.Error(context.TODO(), "❌ 加载集群内配置失败", zap.Error(err))
		return nil, err
	}

	utils.Info(context.TODO(), "✅ 使用集群内配置")
	return cfg, nil
}
