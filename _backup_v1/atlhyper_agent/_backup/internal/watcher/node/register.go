package node

import (
	"AtlHyper/atlhyper_agent/utils"
	"log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 注册器：将 NodeWatcher 注册到 controller-runtime
//
// 执行步骤：
// 1. 从 utils 中获取共享的 Kubernetes 客户端
// 2. 构建 NodeWatcher 实例
// 3. 将其注册到 controller-runtime 的管理器中
// 若注册失败则记录错误日志
func RegisterWatcher(mgr ctrl.Manager) error {
	// 获取全局共享 Kubernetes 客户端
	client := utils.GetClient()

	// 构造 NodeWatcher 实例
	nodeWatcher := NewNodeWatcher(client)

	// 注册到 controller-runtime 管理器
	if err := nodeWatcher.SetupWithManager(mgr); err != nil {
		log.Printf("❌ 注册 NodeWatcher 失败: %v", err)
		return err
	}

	return nil
}

// ✅ 工厂方法：使用注入的 client 构造 NodeWatcher 实例
func NewNodeWatcher(c client.Client) *NodeWatcher {
	return &NodeWatcher{client: c}
}
