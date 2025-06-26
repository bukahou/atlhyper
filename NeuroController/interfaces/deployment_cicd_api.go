// interfaces/deployment_cicd_api.go

package interfaces

import (
	"NeuroController/internal/deployer"
)

// 对外暴露一个封装函数供 external 层调用
func UpdateDeploymentByTag(repo, tag string) error {
	return deployer.UpdateDeploymentByTag(repo, tag)
}
