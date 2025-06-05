// =======================================================================================
// 📄 internal/bootstrap/manager.go
//
// ✨ 功能说明：
//     封装 controller-runtime 的管理器启动逻辑，统一加载所有 Watcher 并启动控制器循环。
//     用作 cmd/neurocontroller/main.go 的核心引导模块，解耦主程序入口与业务注册逻辑。
//
// 📦 提供功能：
//     - StartManager(): 启动 controller-runtime 管理器
//
// 📍 使用场景：
//     - 被 main.go 调用，作为统一启动控制器的入口
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
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

// ✅ 启动控制器管理器（加载所有 Watcher 并运行）
func StartManager() {
	// ✅ 创建 controller-runtime 管理器
	cfg, err := resolveRestConfig()
	if err != nil {
		utils.Fatal(nil, "❌ 获取 Kubernetes 配置失败", zap.Error(err))
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		//后续添加需要监控的NS，暂定全集群监控
		//Namespace: "default",
	})
	if err != nil {
		utils.Fatal(nil, "❌ 初始化 Controller Manager 失败", zap.Error(err))
	}

	// ✅ 注册所有 Watcher
	if err := watcher.RegisterAllWatchers(mgr); err != nil {
		utils.Fatal(nil, "❌ 注册 Watcher 模块失败", zap.Error(err))
	}

	// ✅ 启动控制循环（阻塞）
	utils.Info(nil, "🚀 启动 controller-runtime 管理器中 ...")
	if err := mgr.Start(context.Background()); err != nil {
		utils.Fatal(nil, "❌ 控制器主循环运行失败", zap.Error(err))
	}
}

// ✅ 私有函数：自动判断 kubeconfig / InClusterConfig
func resolveRestConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err == nil {
			utils.Info(context.TODO(), "✅ 使用本地 kubeconfig 启动")
			return cfg, nil
		}
		utils.Warn(context.TODO(), "⚠️ 加载本地 kubeconfig 失败，尝试 InCluster", zap.Error(err))
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		utils.Error(context.TODO(), "❌ 无法加载 InCluster 配置", zap.Error(err))
		return nil, err
	}

	utils.Info(context.TODO(), "✅ 使用集群内配置启动")
	return cfg, nil
}
