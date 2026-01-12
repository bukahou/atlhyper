// sdk/k8s/cluster.go
// ClusterInfo K8s 实现
package k8s

import (
	"context"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type k8sClusterInfo struct {
	coreClient *kubernetes.Clientset
	restConfig *rest.Config

	clusterID     string
	clusterIDOnce sync.Once
}

func newK8sClusterInfo(coreClient *kubernetes.Clientset, restConfig *rest.Config) *k8sClusterInfo {
	return &k8sClusterInfo{
		coreClient: coreClient,
		restConfig: restConfig,
	}
}

func (c *k8sClusterInfo) GetClusterID(ctx context.Context) (string, error) {
	var err error

	c.clusterIDOnce.Do(func() {
		// 获取 kube-system Namespace 的 UID 作为集群 ID
		ns, getErr := c.coreClient.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
		if getErr != nil {
			err = fmt.Errorf("获取 kube-system namespace 失败: %w", getErr)
			c.clusterID = "unknown-cluster"
			return
		}
		c.clusterID = string(ns.UID)
	})

	if err != nil {
		return c.clusterID, err
	}
	return c.clusterID, nil
}

func (c *k8sClusterInfo) HealthCheck(ctx context.Context) error {
	// 尝试获取 API Server 版本信息
	_, err := c.coreClient.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("K8s API Server 健康检查失败: %w", err)
	}
	return nil
}
