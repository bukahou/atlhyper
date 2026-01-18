// Package impl K8sClient 接口的具体实现
//
// networking.go - networkingv1 资源操作
//
// 本文件实现 networkingv1 API 组的资源操作：
//   - Ingress: List, Get
package impl

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/sdk"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// =============================================================================
// Ingress 操作
// =============================================================================

func (c *Client) ListIngresses(ctx context.Context, namespace string, opts sdk.ListOptions) ([]networkingv1.Ingress, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.NetworkingV1().Ingresses(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetIngress(ctx context.Context, namespace, name string) (*networkingv1.Ingress, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
}

// =============================================================================
// NetworkPolicy 操作
// =============================================================================

func (c *Client) ListNetworkPolicies(ctx context.Context, namespace string, opts sdk.ListOptions) ([]networkingv1.NetworkPolicy, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
