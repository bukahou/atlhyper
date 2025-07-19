// =======================================================================================
// 📄 restart.go
//
// ✨ 功能说明：
//     提供基于 Pod 名称的重启功能（实际上是删除操作，由上层控制器自动拉起新副本）
//     用于 UI API 中的“重启 Pod”操作按钮。
//
// 🔁 注意：
//     Kubernetes 中无直接 “restart” Pod 接口，只能通过 Delete 实现重建效果。
//
// 📍 调用链：
//     external → interfaces → internal/operator/pod/RestartPod
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: July 2025
// =======================================================================================

package pod

import (
	"context"
	"fmt"
	"io"
	"strings"

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestartPod 删除指定命名空间下的 Pod，用于模拟“重启”行为。
// Kubernetes 会自动根据控制器（如 Deployment）重建该 Pod。
//
// 参数：
//   - ctx:      上下文，用于链路跟踪 / 超时控制
//   - namespace: Pod 所在命名空间
//   - name:      Pod 的名称
//
// 返回：
//   - error: 若删除失败，返回详细错误信息；否则返回 nil
func RestartPod(ctx context.Context, namespace, name string) error {
	// 获取共享的 client-go CoreV1 客户端
	client := utils.GetCoreClient()

	// 设置删除策略：后台删除，允许调度器立即重新拉起新 Pod
	deletePolicy := metav1.DeletePropagationBackground

	// 设置宽限期：给容器 3 秒优雅退出时间
	gracePeriodSeconds := int64(3)

	// 执行删除操作（模拟“重启”）
	err := client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
		PropagationPolicy:  &deletePolicy,
	})
	if err != nil {
		// 删除失败，返回包装后的错误
		return fmt.Errorf("failed to delete pod %s/%s: %w", namespace, name, err)
	}

	// 成功返回
	return nil
}

// GetPodLogs 获取指定 Pod 的日志信息（支持指定容器与 tailLines）
//
// 参数：
//   - ctx: 上下文，用于链路追踪 / 超时控制
//   - namespace: Pod 所属命名空间
//   - name: Pod 名称
//   - container: 容器名称（可选，若为空则自动判断是否单容器）
//   - tailLines: 获取日志的尾部行数（若 <= 0 则默认 100）
//
// 返回：
//   - string: 日志内容
//   - error: 若失败，返回错误信息
func GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	client := utils.GetCoreClient()

	// 设置默认日志行数（防止无效请求）
	if tailLines <= 0 {
		tailLines = 100
	}

	// 构造日志请求参数
	opts := &corev1.PodLogOptions{
		TailLines:  &tailLines,
		Timestamps: true,
	}

	// 自动判断容器名（仅当未指定且为单容器 Pod）
	if container == "" {
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("无法获取 Pod：%w", err)
		}
		containers := pod.Spec.Containers
		if len(containers) == 1 {
			opts.Container = containers[0].Name
		} else {
			return "", fmt.Errorf("Pod 中存在多个容器，请指定 container 参数")
		}
	} else {
		opts.Container = container
	}

	// 发起日志请求
	req := client.CoreV1().Pods(namespace).GetLogs(name, opts)

	// 获取日志流
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("无法获取日志流 %s/%s: %w", namespace, name, err)
	}
	defer stream.Close()

	// 读取日志内容
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, stream); err != nil {
		return "", fmt.Errorf("读取日志失败: %w", err)
	}

	return buf.String(), nil
}
