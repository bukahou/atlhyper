// Package command 指令执行服务
//
// 本包实现 service.CommandService 接口，负责执行 Master 下发的指令。
//
// 支持的指令类型 (Action):
//   - scale: 扩缩容 Deployment
//   - restart: 重启 Deployment (滚动重启)
//   - update_image: 更新容器镜像
//   - delete: 删除资源 (Pod 或通用资源)
//   - get_logs: 获取 Pod 日志
//   - cordon: 封锁节点
//   - uncordon: 解封节点
//   - dynamic: 动态 API 调用 (AI 只读查询)
//
// 执行流程:
//  1. 根据 Action 分发到对应的 handler
//  2. 解析 Params 中的参数
//  3. 调用 Repository 执行操作
//  4. 封装 Result 返回
package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/service"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/command"
)

// commandService 指令执行服务实现
//
// 核心依赖:
//   - podRepo: Pod 查询 (日志获取)
//   - genericRepo: 所有写操作 + 动态查询
//   - traceQueryRepo: Trace 按需查询 (ClickHouse, 可选)
//   - logQueryRepo: Log 按需查询 (ClickHouse, 可选)
//   - metricsQueryRepo: Metrics 按需查询 (ClickHouse, 可选)
//   - sloQueryRepo: SLO 按需查询 (ClickHouse, 可选)
type commandService struct {
	podRepo     repository.PodRepository
	genericRepo repository.GenericRepository

	// ClickHouse 查询仓库 (可选)
	traceQueryRepo   repository.TraceQueryRepository
	logQueryRepo     repository.LogQueryRepository
	metricsQueryRepo repository.MetricsQueryRepository
	sloQueryRepo     repository.SLOQueryRepository
}

// NewCommandService 创建指令服务
func NewCommandService(
	podRepo repository.PodRepository,
	genericRepo repository.GenericRepository,
	traceQueryRepo repository.TraceQueryRepository,
	logQueryRepo repository.LogQueryRepository,
	metricsQueryRepo repository.MetricsQueryRepository,
	sloQueryRepo repository.SLOQueryRepository,
) service.CommandService {
	return &commandService{
		podRepo:          podRepo,
		genericRepo:      genericRepo,
		traceQueryRepo:   traceQueryRepo,
		logQueryRepo:     logQueryRepo,
		metricsQueryRepo: metricsQueryRepo,
		sloQueryRepo:     sloQueryRepo,
	}
}

// Execute 执行指令
//
// 根据 Command.Action 分发到对应的处理函数。
// 无论成功与否，都返回 Result，不返回 error。
//
// 返回值:
//   - Success=true: 执行成功，Data 包含返回数据 (如日志内容)
//   - Success=false: 执行失败，Error 包含错误信息
func (s *commandService) Execute(ctx context.Context, cmd *command.Command) *command.Result {
	start := time.Now()

	result := &command.Result{
		CommandID:  cmd.ID,
		ExecutedAt: time.Now(),
	}

	var err error
	var data any

	switch cmd.Action {
	case command.ActionScale:
		err = s.handleScale(ctx, cmd)
	case command.ActionRestart:
		err = s.handleRestart(ctx, cmd)
	case command.ActionUpdateImage:
		err = s.handleUpdateImage(ctx, cmd)
	case command.ActionGetLogs:
		data, err = s.handleGetLogs(ctx, cmd)
	case command.ActionGetConfigMap:
		data, err = s.handleGetConfigMap(ctx, cmd)
	case command.ActionGetSecret:
		data, err = s.handleGetSecret(ctx, cmd)
	case command.ActionDynamic:
		data, err = s.handleDynamic(ctx, cmd)
	case command.ActionDelete:
		err = s.handleDelete(ctx, cmd)
	case command.ActionCordon:
		err = s.handleCordon(ctx, cmd)
	case command.ActionUncordon:
		err = s.handleUncordon(ctx, cmd)
	case command.ActionQueryTraces:
		data, err = s.handleQueryTraces(ctx, cmd)
	case command.ActionQueryTraceDetail:
		data, err = s.handleQueryTraceDetail(ctx, cmd)
	case command.ActionQueryLogs:
		data, err = s.handleQueryLogs(ctx, cmd)
	case command.ActionQueryMetrics:
		data, err = s.handleQueryMetrics(ctx, cmd)
	case command.ActionQuerySLO:
		data, err = s.handleQuerySLO(ctx, cmd)
	default:
		err = fmt.Errorf("unknown action: %s", cmd.Action)
	}

	result.ExecTime = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		if data != nil {
			// 将返回数据转换为字符串
			switch v := data.(type) {
			case string:
				result.Output = v
			default:
				// 其他类型 JSON 序列化
				if b, e := json.Marshal(v); e == nil {
					result.Output = string(b)
				}
			}
		}
	}

	return result
}

// handleScale 处理扩缩容指令
func (s *commandService) handleScale(ctx context.Context, cmd *command.Command) error {
	var params struct {
		Replicas int32 `json:"replicas"`
	}
	if err := s.parseParams(cmd.Params, &params); err != nil {
		return fmt.Errorf("invalid scale params: %w", err)
	}

	return s.genericRepo.ScaleDeployment(ctx, cmd.Namespace, cmd.Name, params.Replicas)
}

// handleRestart 处理重启指令
func (s *commandService) handleRestart(ctx context.Context, cmd *command.Command) error {
	return s.genericRepo.RestartDeployment(ctx, cmd.Namespace, cmd.Name)
}

// handleUpdateImage 处理更新镜像指令
func (s *commandService) handleUpdateImage(ctx context.Context, cmd *command.Command) error {
	var params struct {
		Container string `json:"container,omitempty"`
		Image     string `json:"image"`
	}
	if err := s.parseParams(cmd.Params, &params); err != nil {
		return fmt.Errorf("invalid update_image params: %w", err)
	}
	if params.Image == "" {
		return fmt.Errorf("image is required")
	}

	return s.genericRepo.UpdateDeploymentImage(ctx, cmd.Namespace, cmd.Name, params.Container, params.Image)
}

// handleCordon 处理封锁节点指令
func (s *commandService) handleCordon(ctx context.Context, cmd *command.Command) error {
	return s.genericRepo.CordonNode(ctx, cmd.Name)
}

// handleUncordon 处理解封节点指令
func (s *commandService) handleUncordon(ctx context.Context, cmd *command.Command) error {
	return s.genericRepo.UncordonNode(ctx, cmd.Name)
}

// handleGetLogs 处理获取日志指令
func (s *commandService) handleGetLogs(ctx context.Context, cmd *command.Command) (string, error) {
	var params struct {
		Container    string `json:"container,omitempty"`
		TailLines    int64  `json:"tailLines,omitempty"`
		SinceSeconds int64  `json:"sinceSeconds,omitempty"`
		Timestamps   bool   `json:"timestamps,omitempty"`
		Previous     bool   `json:"previous,omitempty"`
	}
	if cmd.Params != nil {
		if err := s.parseParams(cmd.Params, &params); err != nil {
			return "", fmt.Errorf("invalid log params: %w", err)
		}
	}

	// 强制 tailLines 范围
	const maxTailLines int64 = 200
	if params.TailLines <= 0 {
		params.TailLines = 100
	}
	if params.TailLines > maxTailLines {
		params.TailLines = maxTailLines
	}

	// Container 为空时自动选择主容器（避免多容器 Pod 报错）
	if params.Container == "" {
		pod, err := s.podRepo.Get(ctx, cmd.Namespace, cmd.Name)
		if err == nil && len(pod.Containers) > 0 {
			for _, c := range pod.Containers {
				if !model_v3.IsSidecarContainer(c.Name) {
					params.Container = c.Name
					break
				}
			}
			if params.Container == "" {
				params.Container = pod.Containers[0].Name
			}
		}
	}

	return s.podRepo.GetLogs(ctx, cmd.Namespace, cmd.Name, model.LogOptions{
		Container:    params.Container,
		TailLines:    params.TailLines,
		SinceSeconds: params.SinceSeconds,
		Timestamps:   params.Timestamps,
		Previous:     params.Previous,
	})
}

// handleGetConfigMap 处理获取 ConfigMap 数据指令
func (s *commandService) handleGetConfigMap(ctx context.Context, cmd *command.Command) (map[string]string, error) {
	if cmd.Namespace == "" || cmd.Name == "" {
		return nil, fmt.Errorf("namespace and name are required")
	}
	return s.genericRepo.GetConfigMapData(ctx, cmd.Namespace, cmd.Name)
}

// handleGetSecret 处理获取 Secret 数据指令
func (s *commandService) handleGetSecret(ctx context.Context, cmd *command.Command) (map[string]string, error) {
	if cmd.Namespace == "" || cmd.Name == "" {
		return nil, fmt.Errorf("namespace and name are required")
	}
	return s.genericRepo.GetSecretData(ctx, cmd.Namespace, cmd.Name)
}

// handleDynamic 处理 AI 动态查询指令
//
// 将 AI 的高级语义 (command + kind) 翻译为 K8s API 路径，
// 通过 GenericRepository.Execute 执行只读 GET 请求。
//
// Params 格式:
//   - command: get / list / describe / get_events
//   - kind: Pod / Deployment / Node / ...
//   - label_selector: 标签过滤 (list 时使用)
//   - involved_kind: 事件关联资源类型 (get_events 时使用)
//   - involved_name: 事件关联资源名称 (get_events 时使用)
func (s *commandService) handleDynamic(ctx context.Context, cmd *command.Command) (string, error) {
	var params struct {
		Command       string `json:"command"`
		Kind          string `json:"kind"`
		LabelSelector string `json:"label_selector"`
		InvolvedKind  string `json:"involved_kind"`
		InvolvedName  string `json:"involved_name"`
	}
	if err := s.parseParams(cmd.Params, &params); err != nil {
		return "", fmt.Errorf("invalid dynamic params: %w", err)
	}

	if params.Command == "" {
		return "", fmt.Errorf("command is required in params")
	}
	if params.Kind == "" && params.Command != "get_events" {
		return "", fmt.Errorf("kind is required in params")
	}

	// 构建 API 路径
	path, err := buildAPIPath(params.Command, params.Kind, cmd.Namespace, cmd.Name)
	if err != nil {
		return "", fmt.Errorf("build API path: %w", err)
	}

	// 构建查询参数
	query := map[string]string{}
	if params.LabelSelector != "" {
		query["labelSelector"] = params.LabelSelector
	}
	if params.Command == "get_events" {
		if fs := buildEventFieldSelector(params.InvolvedKind, params.InvolvedName); fs != "" {
			query["fieldSelector"] = fs
		}
	}

	// list 类请求强制限制返回数量
	if params.Command == "list" || params.Command == "get_events" {
		if _, hasLimit := query["limit"]; !hasLimit {
			query["limit"] = "200"
		}
	}

	// 执行查询
	resp, err := s.genericRepo.Execute(ctx, &model.DynamicRequest{
		Path:  path,
		Query: query,
	})
	if err != nil {
		return "", fmt.Errorf("execute dynamic query: %w", err)
	}

	// HTTP 4xx/5xx 视为错误
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(resp.Body))
	}

	// AI 来源的 list 操作: 转为表格摘要（大幅压缩体积）
	if params.Command == "list" && cmd.Source == "ai" {
		return summarizeList(stripManagedFields(resp.Body)), nil
	}

	// 其他情况: 保留完整 JSON（仅去 managedFields）
	return string(stripManagedFields(resp.Body)), nil
}

// handleDelete 处理通用删除指令
func (s *commandService) handleDelete(ctx context.Context, cmd *command.Command) error {
	var params struct {
		GracePeriodSeconds *int64 `json:"gracePeriodSeconds,omitempty"`
		Force              bool   `json:"force,omitempty"`
	}
	if cmd.Params != nil {
		if err := s.parseParams(cmd.Params, &params); err != nil {
			return fmt.Errorf("invalid delete params: %w", err)
		}
	}

	opts := model.DeleteOptions{
		GracePeriodSeconds: params.GracePeriodSeconds,
		Force:              params.Force,
	}

	// Pod 使用专门的删除方法
	if cmd.Kind == "Pod" {
		return s.genericRepo.DeletePod(ctx, cmd.Namespace, cmd.Name, opts)
	}

	// 其他资源使用通用删除
	return s.genericRepo.Delete(ctx, cmd.Kind, cmd.Namespace, cmd.Name, opts)
}

// parseParams 解析参数
func (s *commandService) parseParams(params any, target any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// =============================================================================
// Dynamic 查询辅助 (AI 专用)
// =============================================================================

// resourceInfo K8s 资源的 API 路径信息
type resourceInfo struct {
	APIPrefix    string // "/api/v1" 或 "/apis/apps/v1" 等
	Resource     string // 复数小写: "pods", "deployments"
	ClusterScope bool   // 是否集群级资源 (无需 namespace)
}

// kindToResource Kind → API 路径映射表
var kindToResource = map[string]resourceInfo{
	// Core API (/api/v1)
	"Pod":                   {"/api/v1", "pods", false},
	"Node":                  {"/api/v1", "nodes", true},
	"Service":               {"/api/v1", "services", false},
	"ConfigMap":             {"/api/v1", "configmaps", false},
	"Secret":                {"/api/v1", "secrets", false},
	"Event":                 {"/api/v1", "events", false},
	"Namespace":             {"/api/v1", "namespaces", true},
	"PersistentVolume":      {"/api/v1", "persistentvolumes", true},
	"PersistentVolumeClaim": {"/api/v1", "persistentvolumeclaims", false},
	"ServiceAccount":        {"/api/v1", "serviceaccounts", false},
	"ResourceQuota":         {"/api/v1", "resourcequotas", false},
	"LimitRange":            {"/api/v1", "limitranges", false},
	"Endpoints":             {"/api/v1", "endpoints", false},
	// Apps API (/apis/apps/v1)
	"Deployment":  {"/apis/apps/v1", "deployments", false},
	"StatefulSet": {"/apis/apps/v1", "statefulsets", false},
	"DaemonSet":   {"/apis/apps/v1", "daemonsets", false},
	"ReplicaSet":  {"/apis/apps/v1", "replicasets", false},
	// Batch API (/apis/batch/v1)
	"Job":     {"/apis/batch/v1", "jobs", false},
	"CronJob": {"/apis/batch/v1", "cronjobs", false},
	// Networking API (/apis/networking.k8s.io/v1)
	"Ingress":       {"/apis/networking.k8s.io/v1", "ingresses", false},
	"NetworkPolicy": {"/apis/networking.k8s.io/v1", "networkpolicies", false},
	// Autoscaling API (/apis/autoscaling/v2)
	"HorizontalPodAutoscaler": {"/apis/autoscaling/v2", "horizontalpodautoscalers", false},
	"HPA":                     {"/apis/autoscaling/v2", "horizontalpodautoscalers", false},
}

// buildAPIPath 根据 command/kind/namespace/name 构建 K8s API 路径
func buildAPIPath(command, kind, namespace, name string) (string, error) {
	if command == "get_events" {
		kind = "Event"
	}

	info, ok := kindToResource[kind]
	if !ok {
		return "", fmt.Errorf("unsupported kind: %s", kind)
	}

	switch command {
	case "list", "get_events":
		if info.ClusterScope || namespace == "" {
			return fmt.Sprintf("%s/%s", info.APIPrefix, info.Resource), nil
		}
		return fmt.Sprintf("%s/namespaces/%s/%s", info.APIPrefix, namespace, info.Resource), nil

	case "get", "describe":
		if name == "" {
			return "", fmt.Errorf("name is required for %s", command)
		}
		if info.ClusterScope {
			return fmt.Sprintf("%s/%s/%s", info.APIPrefix, info.Resource, name), nil
		}
		if namespace == "" {
			return "", fmt.Errorf("namespace is required for %s %s", command, kind)
		}
		return fmt.Sprintf("%s/namespaces/%s/%s/%s", info.APIPrefix, namespace, info.Resource, name), nil

	default:
		return "", fmt.Errorf("unsupported command: %s", command)
	}
}

// stripManagedFields 从 K8s API 响应中移除 managedFields
// managedFields 仅用于 Server-Side Apply 冲突检测，对 AI 分析无用但占 30-60% 体积
func stripManagedFields(data []byte) []byte {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data // 非 JSON 原样返回
	}
	removeManagedFields(obj)
	result, err := json.Marshal(obj)
	if err != nil {
		return data
	}
	return result
}

// removeManagedFields 递归移除 metadata.managedFields
func removeManagedFields(obj interface{}) {
	switch v := obj.(type) {
	case map[string]interface{}:
		if meta, ok := v["metadata"].(map[string]interface{}); ok {
			delete(meta, "managedFields")
		}
		// 递归处理 items（list 返回的数组）
		if items, ok := v["items"].([]interface{}); ok {
			for _, item := range items {
				removeManagedFields(item)
			}
		}
	}
}

// buildEventFieldSelector 构建 Event 查询的 fieldSelector
func buildEventFieldSelector(involvedKind, involvedName string) string {
	var selectors []string
	if involvedKind != "" {
		selectors = append(selectors, "involvedObject.kind="+involvedKind)
	}
	if involvedName != "" {
		selectors = append(selectors, "involvedObject.name="+involvedName)
	}
	return strings.Join(selectors, ",")
}
