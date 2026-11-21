// Package k8s K8sClient 接口的具体实现
//
// apps.go - appsv1 资源操作
//
// 本文件实现 appsv1 API 组的资源操作：
//   - Deployment: List, Get, Scale, Restart, UpdateImage
//   - StatefulSet: List, Get
//   - DaemonSet: List, Get
//   - ReplicaSet: List
package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// =============================================================================
// Deployment 操作
// =============================================================================

func (c *Client) ListDeployments(ctx context.Context, namespace string, opts sdk.ListOptions) ([]appsv1.Deployment, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

// UpdateDeploymentScale 更新 Deployment 副本数
//
// 通过 Scale 子资源 API 更新副本数
func (c *Client) UpdateDeploymentScale(ctx context.Context, namespace, name string, replicas int32) error {
	scale, err := c.clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scale.Spec.Replicas = replicas
	_, err = c.clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return err
}

// RestartDeployment 重启 Deployment
//
// 通过修改 Pod Template 的 annotation 触发滚动更新
func (c *Client) RestartDeployment(ctx context.Context, namespace, name string) error {
	patchData, err := restartPatch(time.Now())
	if err != nil {
		return err
	}
	_, err = c.clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
	return err
}

// UpdateDeploymentImage 更新 Deployment 指定容器的镜像，触发滚动更新
//
// container 必填：改哪个容器由调用方明确指定（参数在 command.UpdateImageParams 里校验）
func (c *Client) UpdateDeploymentImage(ctx context.Context, namespace, name, container, image string) error {
	deploy, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	found := false
	for _, cont := range deploy.Spec.Template.Spec.Containers {
		if cont.Name == container {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("container %s not found in deployment %s", container, name)
	}

	patchData, err := imagePatch(container, image)
	if err != nil {
		return err
	}
	_, err = c.clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
	return err
}

// imagePatch 生成只改一个容器镜像的 strategic merge patch（按容器名合并）。
// 用 json.Marshal 生成，任何字段内容都只会是 JSON 字符串值，⛔ 不拼接。
func imagePatch(container, image string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []map[string]string{{"name": container, "image": image}},
				},
			},
		},
	})
}

// restartPatch 生成滚动重启用的 patch：改 Pod 模板的注解（与 kubectl rollout restart 相同的键）
func restartPatch(at time.Time) ([]byte, error) {
	return json.Marshal(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{"kubectl.kubernetes.io/restartedAt": at.Format(time.RFC3339)},
				},
			},
		},
	})
}

// =============================================================================
// StatefulSet 操作
// =============================================================================

func (c *Client) ListStatefulSets(ctx context.Context, namespace string, opts sdk.ListOptions) ([]appsv1.StatefulSet, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// =============================================================================
// DaemonSet 操作
// =============================================================================

func (c *Client) ListDaemonSets(ctx context.Context, namespace string, opts sdk.ListOptions) ([]appsv1.DaemonSet, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.AppsV1().DaemonSets(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetDaemonSet(ctx context.Context, namespace, name string) (*appsv1.DaemonSet, error) {
	return c.clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// =============================================================================
// ReplicaSet 操作
// =============================================================================

func (c *Client) ListReplicaSets(ctx context.Context, namespace string, opts sdk.ListOptions) ([]appsv1.ReplicaSet, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.AppsV1().ReplicaSets(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
