package pod

import (
	"AtlHyper/atlhyper_agent/utils"
	"log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 注册器：将 PodWatcher 注册到 controller-runtime
//
// 获取全局 Kubernetes 客户端 → 构造 watcher 实例 →
// 注册到 controller-runtime 的管理器中。
// 若注册失败，则记录错误日志。
func RegisterWatcher(mgr ctrl.Manager) error {
	// 获取共享的 Kubernetes 客户端（通过 utils 封装）
	client := utils.GetClient()

	// 注入客户端并构造 PodWatcher 实例
	podWatcher := NewPodWatcher(client)

	// 注册到管理器
	if err := podWatcher.SetupWithManager(mgr); err != nil {
		log.Printf("❌ 注册 PodWatcher 失败: %v", err)
		return err
	}

	return nil
}

// ✅ 工厂函数：使用注入的 client 创建 PodWatcher 实例
func NewPodWatcher(c client.Client) *PodWatcher {
	return &PodWatcher{client: c}
}
