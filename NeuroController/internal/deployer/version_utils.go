package deployer

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
)

// GetCurrentVersion 从注解中读取当前使用的版本
func GetCurrentVersion(deploy *appsv1.Deployment) string {
	if deploy.Annotations == nil {
		return ""
	}
	return deploy.Annotations["neurocontroller.version.latest"]
}

// GetPreviousVersion 从注解中读取上一个使用的版本
func GetPreviousVersion(deploy *appsv1.Deployment) string {
	if deploy.Annotations == nil {
		return ""
	}
	return deploy.Annotations["neurocontroller.version.previous"]
}

// UpdateVersionAnnotations 同步更新注解字段
func UpdateVersionAnnotations(deploy *appsv1.Deployment, newVersion string) {
	if deploy.Annotations == nil {
		deploy.Annotations = map[string]string{}
	}
	deploy.Annotations["neurocontroller.version.previous"] = deploy.Annotations["neurocontroller.version.latest"]
	deploy.Annotations["neurocontroller.version.latest"] = newVersion
}

// RollbackToPreviousVersion 将 Deployment 回滚到上一个版本
func RollbackToPreviousVersion(deploy *appsv1.Deployment) (string, error) {
	prev := GetPreviousVersion(deploy)
	if prev == "" {
		return "", fmt.Errorf("找不到 previous 版本，无法回滚")
	}

	// 更新 image
	if len(deploy.Spec.Template.Spec.Containers) == 0 {
		return "", fmt.Errorf("Deployment 不包含容器，无法设置镜像")
	}
	deploy.Spec.Template.Spec.Containers[0].Image = prev

	// 更新 latest 注解（previous 保持不变）
	if deploy.Annotations == nil {
		deploy.Annotations = map[string]string{}
	}
	deploy.Annotations["neurocontroller.version.latest"] = prev

	return prev, nil
}

// BuildImageFrom 拼接镜像名：repo:tag
func BuildImageFrom(repo, tag string) string {
	return fmt.Sprintf("%s:%s", repo, tag)
}

// IsSameVersion 判断版本是否一致（用于跳过更新）
func IsSameVersion(current, incoming string) bool {
	return current == incoming
}
