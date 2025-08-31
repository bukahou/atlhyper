package endpoint

import (
	"AtlHyper/atlhyper_agent/utils"
	"log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 将 EndpointWatcher 注册到 controller-runtime 管理器中
func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	watcher := NewEndpointWatcher(client)

	if err := watcher.SetupWithManager(mgr); err != nil {
		log.Printf("❌ EndpointWatcher 注册失败: %v", err)
		return err
	}

	return nil
}

// ✅ 构造一个新的 EndpointWatcher 实例
func NewEndpointWatcher(c client.Client) *EndpointWatcher {
	return &EndpointWatcher{client: c}
}
