package uiapi

// =======================================================================================
// 📄 pod_api.go（interfaces/ui_api）
//
// ✨ 文件功能说明：
//     提供 Pod 相关的 REST 接口：列表获取、状态统计、资源用量聚合等。
//     供前端 UI 页面如 Pod 面板、命名空间视图、集群概览使用。
//
// 📦 依赖模块：
//     - internal/query/pod：获取 Pod 资源与使用量
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

import (
	operatorpod "NeuroController/internal/operator/pod"
	"NeuroController/internal/query/pod"
	"context"

	corev1 "k8s.io/api/core/v1"
)

// GetAllPods 获取所有命名空间下的 Pod 列表
func GetAllPods(ctx context.Context) ([]corev1.Pod, error) {
	return pod.ListAllPods(ctx)
}

// GetPodsByNamespace 获取指定命名空间下的 Pod 列表
func GetPodsByNamespace(ctx context.Context, ns string) ([]corev1.Pod, error) {
	return pod.ListPodsByNamespace(ctx, ns)
}

// GetPodStatusSummary 获取所有 Pod 的状态统计
func GetPodStatusSummary(ctx context.Context) (*pod.PodSummary, error) {
	pods, err := pod.ListAllPods(ctx)
	if err != nil {
		return nil, err
	}
	summary := pod.SummarizePodsByStatus(pods)
	return &summary, nil
}

// GetPodUsages 获取所有 Pod 的资源使用情况
func GetPodUsages(ctx context.Context) ([]pod.PodUsage, error) {
	return pod.ListAllPodUsages(ctx)
}

// GetAllPodInfos 获取所有 Pod 的精简信息（供 UI 展示使用）
func GetAllPodInfos(ctx context.Context) ([]pod.PodInfo, error) {
	return pod.ListAllPodInfos(ctx)
}

// GetPodDescribe 获取指定 Pod 的详细信息（结构体中包含 Pod 本体与 Events）
func GetPodDescribe(ctx context.Context, namespace, name string) (*pod.PodDescribeInfo, error) {
	return pod.GetPodDescribeInfo(ctx, namespace, name)
}

// ============================================================================================================================================
// ============================================================================================================================================
// 操作函数
// ============================================================================================================================================
// ============================================================================================================================================

// RestartPod 重启指定命名空间下的 Pod（实际上为删除操作，由控制器自动拉起新副本）
// 用于 UI 操作按钮「重启 Pod」调用。
//
// 参数：
//   - ctx: 上下文，用于链路追踪 / 超时控制
//   - namespace: Pod 所属命名空间
//   - name: Pod 名称
//
// 返回：
//   - error: 若删除失败，返回错误；成功返回 nil
func RestartPod(ctx context.Context, namespace, name string) error {
	return operatorpod.RestartPod(ctx, namespace, name)
}

// GetPodLogs 获取指定 Pod 中某个容器的日志尾部内容（默认容器为空则使用首个）
//
// 参数：
//   - ctx: 上下文
//   - namespace: Pod 所属命名空间
//   - name: Pod 名称
//   - container: 容器名称（可选）
//   - tailLines: 获取尾部日志行数（例如 100）
//
// 返回：
//   - string: 日志内容（纯文本）
//   - error: 若获取失败，返回错误信息
func GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	return operatorpod.GetPodLogs(ctx, namespace, name, container, tailLines)
}
