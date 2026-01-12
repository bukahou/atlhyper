// sdk/k8s/operators.go
// Pod/Node/Deployment 操作实现
package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"AtlHyper/atlhyper_agent/sdk"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ==================== Pod Operator ====================

type k8sPodOperator struct {
	client *kubernetes.Clientset
}

func newK8sPodOperator(client *kubernetes.Clientset) *k8sPodOperator {
	return &k8sPodOperator{client: client}
}

func (o *k8sPodOperator) RestartPod(ctx context.Context, key sdk.ObjectKey) error {
	deletePolicy := metav1.DeletePropagationBackground
	gracePeriodSeconds := int64(3)

	err := o.client.CoreV1().Pods(key.Namespace).Delete(ctx, key.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
		PropagationPolicy:  &deletePolicy,
	})
	if err != nil {
		return fmt.Errorf("删除 Pod %s/%s 失败: %w", key.Namespace, key.Name, err)
	}
	return nil
}

func (o *k8sPodOperator) GetPodLogs(ctx context.Context, key sdk.ObjectKey, opts sdk.LogOptions) (string, error) {
	tailLines := opts.TailLines
	if tailLines <= 0 {
		tailLines = 100
	}

	logOpts := &corev1.PodLogOptions{
		TailLines:  &tailLines,
		Timestamps: opts.Timestamps,
	}

	// 自动判断容器名
	if opts.Container == "" {
		pod, err := o.client.CoreV1().Pods(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("获取 Pod 失败: %w", err)
		}
		if len(pod.Spec.Containers) == 1 {
			logOpts.Container = pod.Spec.Containers[0].Name
		} else {
			return "", fmt.Errorf("Pod 中存在多个容器，请指定 container 参数")
		}
	} else {
		logOpts.Container = opts.Container
	}

	req := o.client.CoreV1().Pods(key.Namespace).GetLogs(key.Name, logOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("获取日志流失败 %s/%s: %w", key.Namespace, key.Name, err)
	}
	defer stream.Close()

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, stream); err != nil {
		return "", fmt.Errorf("读取日志失败: %w", err)
	}

	return buf.String(), nil
}

// ==================== Node Operator ====================

type k8sNodeOperator struct {
	coreClient    *kubernetes.Clientset
	runtimeClient client.Client
}

func newK8sNodeOperator(coreClient *kubernetes.Clientset, runtimeClient client.Client) *k8sNodeOperator {
	return &k8sNodeOperator{
		coreClient:    coreClient,
		runtimeClient: runtimeClient,
	}
}

func (o *k8sNodeOperator) CordonNode(ctx context.Context, name string) error {
	return o.setNodeSchedulable(ctx, name, true)
}

func (o *k8sNodeOperator) UncordonNode(ctx context.Context, name string) error {
	return o.setNodeSchedulable(ctx, name, false)
}

func (o *k8sNodeOperator) setNodeSchedulable(ctx context.Context, nodeName string, unschedulable bool) error {
	var node corev1.Node
	if err := o.runtimeClient.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}

	node.Spec.Unschedulable = unschedulable

	if err := o.runtimeClient.Update(ctx, &node); err != nil {
		return fmt.Errorf("更新节点调度状态失败: %w", err)
	}

	return nil
}

// ==================== Deployment Operator ====================

type k8sDeploymentOperator struct {
	client *kubernetes.Clientset
}

func newK8sDeploymentOperator(client *kubernetes.Clientset) *k8sDeploymentOperator {
	return &k8sDeploymentOperator{client: client}
}

func (o *k8sDeploymentOperator) ScaleDeployment(ctx context.Context, key sdk.ObjectKey, replicas int32) error {
	patch := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, replicas))

	_, err := o.client.AppsV1().Deployments(key.Namespace).Patch(
		ctx,
		key.Name,
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("扩缩容 Deployment 失败: %w", err)
	}

	return nil
}

func (o *k8sDeploymentOperator) UpdateDeploymentImage(ctx context.Context, key sdk.ObjectKey, newImage string) error {
	deploy, err := o.client.AppsV1().Deployments(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	type containerPatch struct {
		Name  string `json:"name"`
		Image string `json:"image"`
	}
	type patchSpec struct {
		Spec struct {
			Template struct {
				Spec struct {
					Containers []containerPatch `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}

	var patch patchSpec
	for _, c := range deploy.Spec.Template.Spec.Containers {
		patch.Spec.Template.Spec.Containers = append(patch.Spec.Template.Spec.Containers, containerPatch{
			Name:  c.Name,
			Image: newImage,
		})
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %w", err)
	}

	_, err = o.client.AppsV1().Deployments(key.Namespace).Patch(
		ctx,
		key.Name,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("更新 Deployment 镜像失败: %w", err)
	}

	return nil
}
