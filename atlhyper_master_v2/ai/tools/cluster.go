package tools

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/command"
)

// query_cluster 能做的动作与能查的资源类型：允许名单，只在这里定义一份。
// 发给模型的 schema 枚举来自这里，服务端按同一份 schema 校验 ⇒ 名单外的一律在执行前拦下。
var (
	queryClusterActions = []string{"get", "list", "describe", "get_logs", "get_events", "get_configmap"}
	// 与 Agent 的资源映射一致（atlhyper_agent_v2/service/command），⛔ 不含 Secret
	queryClusterKinds = []string{
		"Pod", "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet", "Job", "CronJob",
		"Service", "Endpoints", "Ingress", "NetworkPolicy", "HorizontalPodAutoscaler",
		"ConfigMap", "PersistentVolumeClaim", "PersistentVolume", "ServiceAccount",
		"ResourceQuota", "LimitRange", "Node", "Namespace", "Event",
	}
	// 系统命名空间：显式指定时拒绝
	queryClusterDeniedNamespaces = []string{"kube-system", "kube-public", "kube-node-lease"}
)

const (
	defaultTailLines = 100
	maxTailLines     = 1000
)

type queryClusterParams struct {
	Action        string `json:"action" jsonschema:"操作：get / list / describe 读资源；get_logs 读容器日志；get_events 读最近约 1 小时的 K8s 事件；get_configmap 读 ConfigMap 内容"`
	Kind          string `json:"kind" jsonschema:"资源类型（按 Kubernetes 的写法，如 Pod、Deployment、PersistentVolumeClaim）"`
	Namespace     string `json:"namespace,omitempty" jsonschema:"命名空间。list 可不填以查询全部命名空间；get / describe / get_logs / get_configmap 需要填写；集群级资源（Node、PersistentVolume、Namespace）不填"`
	Name          string `json:"name,omitempty" jsonschema:"资源名称（list 与 get_events 可不填）"`
	LabelSelector string `json:"label_selector,omitempty" jsonschema:"标签选择器，如 app=nginx（list 时过滤）"`
	Container     string `json:"container,omitempty" jsonschema:"容器名称（get_logs 时多容器 Pod 需要指定）"`
	TailLines     int    `json:"tail_lines,omitempty" jsonschema:"返回日志的尾行数（get_logs 用，不填为 100）"`
	InvolvedKind  string `json:"involved_kind,omitempty" jsonschema:"关联资源类型（get_events 时过滤，如 Pod、Node、Deployment）"`
	InvolvedName  string `json:"involved_name,omitempty" jsonschema:"关联资源名称（get_events 时过滤）"`
}

// queryClusterPermission 按动作与资源类型细分权限
func queryClusterPermission(p queryClusterParams) string {
	switch p.Action {
	case "get_logs":
		return toolkit.PermPodLogsRead
	case "get_configmap":
		return toolkit.PermSecretRead
	case "get_events":
		return toolkit.PermEventsRead
	default: // get / list / describe：返回原始对象；ConfigMap 的对象就是它的内容
		if p.Kind == "ConfigMap" {
			return toolkit.PermSecretRead
		}
		return toolkit.PermWorkloadEnvRead
	}
}

func queryClusterTool(d Deps) *toolkit.Tool {
	return toolkit.MustDefine(toolkit.Spec[queryClusterParams]{
		Name: "query_cluster",
		Description: "直连 API Server 读取 Kubernetes 集群的实时数据（只读）：资源对象、容器日志、最近的事件。" +
			"排查具体资源的当前状态时用；同一回复里可以并行调用多次。",
		Permission:    toolkit.PermWorkloadEnvRead,
		PermissionFor: queryClusterPermission,
		Permissions:   []string{toolkit.PermWorkloadEnvRead, toolkit.PermPodLogsRead, toolkit.PermSecretRead, toolkit.PermEventsRead},
		Refine: func(s *jsonschema.Schema) {
			s.Properties["action"].Enum = anySlice(queryClusterActions)
			s.Properties["kind"].Enum = anySlice(queryClusterKinds)
			s.Properties["tail_lines"].Minimum = ptr(1.0)
			s.Properties["tail_lines"].Maximum = ptr(float64(maxTailLines))
		},
		Handler: func(ctx context.Context, clusterID string, p queryClusterParams) (string, error) {
			if slices.Contains(queryClusterDeniedNamespaces, p.Namespace) {
				return "", fmt.Errorf("命名空间 %s 是系统命名空间，AI 不允许访问", p.Namespace)
			}
			action, params, err := queryClusterCommand(p)
			if err != nil {
				return "", err
			}
			return runCommand(ctx, d, &model.CreateCommandRequest{
				ClusterID:       clusterID,
				Action:          action,
				TargetKind:      p.Kind,
				TargetNamespace: p.Namespace,
				TargetName:      p.Name,
				Source:          command.SourceAI,
				Params:          params,
			})
		},
	})
}

// queryClusterCommand 工具参数 → 下发给 Agent 的指令：get_logs / get_configmap 有专用动作，其余走 dynamic
func queryClusterCommand(p queryClusterParams) (string, map[string]any, error) {
	switch p.Action {
	case "get", "describe", "get_logs", "get_configmap":
		if p.Name == "" {
			return "", nil, fmt.Errorf("%s 需要填写 name", p.Action)
		}
	}
	switch p.Action {
	case "get_logs":
		tail := p.TailLines
		if tail == 0 {
			tail = defaultTailLines
		}
		return command.ActionGetLogs, command.GetLogsParams{Container: p.Container, TailLines: int64(tail)}.ToParams(), nil
	case "get_configmap":
		return command.ActionGetConfigMap, map[string]any{}, nil
	default:
		params := map[string]any{"command": p.Action, "kind": p.Kind}
		if p.LabelSelector != "" {
			params["label_selector"] = p.LabelSelector
		}
		if p.InvolvedKind != "" {
			params["involved_kind"] = p.InvolvedKind
		}
		if p.InvolvedName != "" {
			params["involved_name"] = p.InvolvedName
		}
		return command.ActionDynamic, params, nil
	}
}

// runCommand 下发指令并等结果；Agent 报告失败时返回错误（⛔ 不当作正常结果）
func runCommand(ctx context.Context, d Deps, req *model.CreateCommandRequest) (string, error) {
	result, err := d.Commands.ExecuteCommandSync(ctx, req, d.ToolTimeout)
	if err != nil {
		return "", fmt.Errorf("执行指令失败: %w", err)
	}
	if !result.Success {
		return "", fmt.Errorf("执行失败: %s", result.Error)
	}
	return result.Output, nil
}

func anySlice(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func ptr[T any](v T) *T { return &v }
