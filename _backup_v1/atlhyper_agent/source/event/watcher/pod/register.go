// source/watcher/pod/register.go
// Pod Watcher 注册
package pod

import (
	"log"

	"AtlHyper/atlhyper_agent/sdk"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegisterWatcher 将 PodWatcher 注册到 controller-runtime
func RegisterWatcher(mgr ctrl.Manager) error {
	// 获取共享的 Kubernetes 客户端
	client := sdk.Get().RuntimeClient()

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
