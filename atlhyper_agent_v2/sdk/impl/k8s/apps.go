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
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err := c.clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	return err
}

// UpdateDeploymentImage 更新 Deployment 容器镜像
//
// 更新指定容器的镜像，触发滚动更新
// container 为空时更新第一个容器
func (c *Client) UpdateDeploymentImage(ctx context.Context, namespace, name, container, image string) error {
	deploy, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	containers := deploy.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		return fmt.Errorf("deployment %s has no containers", name)
	}

	targetIndex := 0
	if container != "" {
		found := false
		for i, cont := range containers {
			if cont.Name == container {
				targetIndex = i
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("container %s not found in deployment %s", container, name)
		}
	}

	patchData := fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"%s","image":"%s"}]}}}}`,
		containers[targetIndex].Name, image)

	_, err = c.clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	return err
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
