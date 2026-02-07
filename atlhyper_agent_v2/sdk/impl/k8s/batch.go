// Package k8s K8sClient 接口的具体实现
//
// batch.go - batchv1 资源操作
//
// 本文件实现 batchv1 API 组的资源操作：
//   - Job: List, Get
//   - CronJob: List, Get
package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/sdk"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// =============================================================================
// Job 操作
// =============================================================================

func (c *Client) ListJobs(ctx context.Context, namespace string, opts sdk.ListOptions) ([]batchv1.Job, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.BatchV1().Jobs(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetJob(ctx context.Context, namespace, name string) (*batchv1.Job, error) {
	return c.clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

// =============================================================================
// CronJob 操作
// =============================================================================

func (c *Client) ListCronJobs(ctx context.Context, namespace string, opts sdk.ListOptions) ([]batchv1.CronJob, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	}
	list, err := c.clientset.BatchV1().CronJobs(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) GetCronJob(ctx context.Context, namespace, name string) (*batchv1.CronJob, error) {
	return c.clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
}
