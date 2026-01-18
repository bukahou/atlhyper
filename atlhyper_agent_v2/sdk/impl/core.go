// Package impl K8sClient 接口的具体实现
//
// core.go - corev1 资源操作
//
// 本文件实现 corev1 API 组的资源操作：
//   - Pod: List, Get, Delete, GetLogs
//   - Node: List, Get, Cordon, Uncordon
//   - Service: List, Get
//   - ConfigMap: List, Get
//   - Secret: List, Get
//   - Namespace: List, Get
//   - Event: List
//   - PersistentVolume: List
//   - PersistentVolumeClaim: List
package impl

import (
	"bytes"
	"context"
	"io"

	"AtlHyper/atlhyper_agent_v2/sdk"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// =============================================================================
// Pod 操作
// =============================================================================

func (c *Client) ListPods(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.Pod, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().Pods(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) DeletePod(ctx context.Context, namespace, name string, opts sdk.DeleteOptions) error {
	deleteOpts := metav1.DeleteOptions{}
	if opts.GracePeriodSeconds != nil {
		deleteOpts.GracePeriodSeconds = opts.GracePeriodSeconds
	}
	return c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, deleteOpts)
}

// GetPodLogs 获取 Pod 日志
//
// 通过流式读取获取 Pod 容器日志
func (c *Client) GetPodLogs(ctx context.Context, namespace, name string, opts sdk.LogOptions) (string, error) {
	podLogOpts := &corev1.PodLogOptions{
		Container:  opts.Container,
		Timestamps: opts.Timestamps,
		Previous:   opts.Previous,
	}
	if opts.TailLines > 0 {
		podLogOpts.TailLines = &opts.TailLines
	}
	if opts.SinceSeconds > 0 {
		podLogOpts.SinceSeconds = &opts.SinceSeconds
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(name, podLogOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// =============================================================================
// Node 操作
// =============================================================================

func (c *Client) ListNodes(ctx context.Context, opts sdk.ListOptions) ([]corev1.Node, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().Nodes().List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	return c.clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

// CordonNode 封锁节点
//
// 通过设置 Unschedulable=true 阻止新 Pod 调度到该节点
func (c *Client) CordonNode(ctx context.Context, name string) error {
	patchData := []byte(`{"spec":{"unschedulable":true}}`)
	_, err := c.clientset.CoreV1().Nodes().Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
	return err
}

// UncordonNode 解封节点
//
// 通过设置 Unschedulable=false 允许新 Pod 调度到该节点
func (c *Client) UncordonNode(ctx context.Context, name string) error {
	patchData := []byte(`{"spec":{"unschedulable":false}}`)
	_, err := c.clientset.CoreV1().Nodes().Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
	return err
}

// =============================================================================
// Service 操作
// =============================================================================

func (c *Client) ListServices(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.Service, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().Services(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
}

// =============================================================================
// ConfigMap 操作
// =============================================================================

func (c *Client) ListConfigMaps(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.ConfigMap, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().ConfigMaps(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
}

// =============================================================================
// Secret 操作
// =============================================================================

func (c *Client) ListSecrets(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.Secret, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().Secrets(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	return c.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// =============================================================================
// Namespace 操作
// =============================================================================

func (c *Client) ListNamespaces(ctx context.Context, opts sdk.ListOptions) ([]corev1.Namespace, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().Namespaces().List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

// =============================================================================
// Event 操作
// =============================================================================

func (c *Client) ListEvents(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.Event, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().Events(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// =============================================================================
// PV/PVC 操作
// =============================================================================

func (c *Client) ListPersistentVolumes(ctx context.Context, opts sdk.ListOptions) ([]corev1.PersistentVolume, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().PersistentVolumes().List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListPersistentVolumeClaims(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.PersistentVolumeClaim, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// =============================================================================
// ResourceQuota 操作
// =============================================================================

func (c *Client) ListResourceQuotas(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.ResourceQuota, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().ResourceQuotas(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// =============================================================================
// LimitRange 操作
// =============================================================================

func (c *Client) ListLimitRanges(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.LimitRange, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().LimitRanges(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// =============================================================================
// ServiceAccount 操作
// =============================================================================

func (c *Client) ListServiceAccounts(ctx context.Context, namespace string, opts sdk.ListOptions) ([]corev1.ServiceAccount, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.CoreV1().ServiceAccounts(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
