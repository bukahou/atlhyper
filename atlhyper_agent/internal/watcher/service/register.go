package service

import (
	"AtlHyper/atlhyper_agent/utils"
	"log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 注册器：将 ServiceWatcher 注册到 controller-runtime
func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	serviceWatcher := NewServiceWatcher(client)

	if err := serviceWatcher.SetupWithManager(mgr); err != nil {
		log.Printf("❌ 注册 ServiceWatcher 失败: %v", err)
		return err
	}

	return nil
}

// ✅ 工厂函数：使用注入的 client 创建 ServiceWatcher 实例
func NewServiceWatcher(c client.Client) *ServiceWatcher {
	return &ServiceWatcher{client: c}
}
