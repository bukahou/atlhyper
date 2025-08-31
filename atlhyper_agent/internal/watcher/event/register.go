package event

import (
	"AtlHyper/atlhyper_agent/utils"
	"log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 将 EventWatcher 注册到 controller-runtime 的管理器中
func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	eventWatcher := NewEventWatcher(client)

	if err := eventWatcher.SetupWithManager(mgr); err != nil {
		log.Printf("❌ 注册 EventWatcher 失败 → %v", err)
		return err
	}

	return nil
}

// ✅ 工厂方法：使用注入的 client 创建新的 EventWatcher 实例
func NewEventWatcher(c client.Client) *EventWatcher {
	return &EventWatcher{client: c}
}
