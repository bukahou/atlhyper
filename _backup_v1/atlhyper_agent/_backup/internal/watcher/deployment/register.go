package deployment

import (
	"AtlHyper/atlhyper_agent/utils"
	"log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 工厂方法：使用共享客户端创建 DeploymentWatcher 实例
func NewDeploymentWatcher(c client.Client) *DeploymentWatcher {
	return &DeploymentWatcher{client: c}
}

func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	deploymentWatcher := NewDeploymentWatcher(client)

	if err := deploymentWatcher.SetupWithManager(mgr); err != nil {
		log.Printf("❌ 注册 DeploymentWatcher 失败: %v", err)
		return err
	}

	return nil
}
