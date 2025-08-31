package node

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent/utils"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// 通用方法：修改节点的调度状态
func SetNodeSchedulable(nodeName string, unschedulable bool) error {
	k8sClient := utils.GetClient()
	ctx := context.TODO()

	var node corev1.Node
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}

	node.Spec.Unschedulable = unschedulable

	if err := k8sClient.Update(ctx, &node); err != nil {
		return fmt.Errorf("更新节点调度状态失败: %w", err)
	}

	return nil
}
