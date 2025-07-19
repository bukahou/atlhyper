package deployer

import (
	"context"
	"fmt"
	"strings"

	"NeuroController/internal/utils"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateDeploymentByTag 根据 tag（如 default-media-v1.0.1）定位 Deployment 并更新镜像及注解
func UpdateDeploymentByTag(repo string, tag string) error {
	ctx := context.Background()
	k8sClient := utils.GetClient()

	// 1️⃣ 从 tag 中解析 ns / deployment / version
	namespace, deploymentName, _, err := parseTagToNSNameVersion(tag)
	if err != nil {
		return fmt.Errorf("解析 tag 失败: %w", err)
	}

	// 2️⃣ 获取 Deployment
	var deploy appsv1.Deployment
	if err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      deploymentName,
	}, &deploy); err != nil {
		return fmt.Errorf("获取 Deployment 失败 [%s/%s]: %w", namespace, deploymentName, err)
	}

	// 3️⃣ 检查容器
	if len(deploy.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("Deployment [%s/%s] 不包含容器", namespace, deploymentName)
	}

	// 4️⃣ 构建完整镜像名（来自 webhook 中的 repo + tag）
	fullImage := fmt.Sprintf("%s:%s", repo, tag)
	currentImage := deploy.Spec.Template.Spec.Containers[0].Image

	// 5️⃣ 若镜像相同则跳过
	if IsSameVersion(currentImage, fullImage) {
		return fmt.Errorf("镜像已是最新版本，无需更新: %s", fullImage)
	}

	// 6️⃣ 设置新镜像
	deploy.Spec.Template.Spec.Containers[0].Image = fullImage

	// 7️⃣ 同步更新注解（latest / previous）
	UpdateVersionAnnotations(&deploy, fullImage)

	// 8️⃣ 提交变更
	if err := k8sClient.Update(ctx, &deploy); err != nil {
		return fmt.Errorf("更新 Deployment 失败: %w", err)
	}

	return nil
}

// parseTagToNSNameVersion 将 tag 拆分为 namespace / deployment / version
// 例如 tag: default-media-v1.0.1 → default / media / v1.0.1
func parseTagToNSNameVersion(tag string) (string, string, string, error) {
	parts := strings.SplitN(tag, "-", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("tag 格式非法，应为 <namespace>-<deployment>-<version>")
	}
	return parts[0], parts[1], parts[2], nil
}
