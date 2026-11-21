// Package toolkit AI 工具的注册表：每个工具 = 名称 + 说明 + 参数结构体 + 权限 + 处理函数，只在一处定义。
//
// 发给模型的参数 schema 由参数结构体生成；执行前用同一份 schema 校验模型给的参数，再解码进结构体 ——
// 告诉模型的约束与服务端执行的约束是同一份（此前 enum 只发给模型，服务端不校验）。
// 发给模型的工具清单 = 已注册 ∩ 调用者有权限：没注册（如没配置 GitHub）的工具不会出现在清单里。
package toolkit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// PermissionCheck 判断会看到工具结果的人是否拥有某项权限
type PermissionCheck func(permission string) bool

// 工具读取的数据所需权限。权限名沿用网关权限表的字符串，是两边的契约（网关测试核对它们都在权限表里）。
// 原则：不能经 AI 拿到网关上拿不到的东西。
const (
	PermNone            = ""                    // 不需要额外权限（能进对话 / 分析即可）
	PermWorkloadEnvRead = "workload.env.read"   // 原始资源对象（含 env 字面值）
	PermEventsRead      = "events.read"         // K8s 事件
	PermSecretRead      = "secret.read"         // ConfigMap 内容（网关上与 Secret 同档）
	PermPodLogsRead     = "pod.logs.read"       // 容器日志
	PermAIOpsRead       = "aiops.read"          // 风险、事件、实体详情
	PermAIAnalyze       = "ai.analyze"          // 触发事件 AI 总结
	PermTracesRead      = "traces.read"         // 链路
	PermLogsRead        = "logs.read"           // OTel 日志正文
	PermSLORead         = "slo.read"            // SLO
	PermDeployHistory   = "deploy.history.read" // 部署历史
)

// 调用失败的分类（调用方按类型判断，⛔ 不比较文案）
var (
	ErrUnknownTool      = errors.New("工具未注册")
	ErrPermissionDenied = errors.New("权限不足")
	ErrInvalidParams    = errors.New("参数不合法")
)

// Spec 定义一个工具。P 是参数结构体：字段的 json 标签是参数名，没有 omitempty 的字段是必填，
// jsonschema 标签是给模型看的说明；枚举、取值范围等标签写不了的约束在 Refine 里补。
type Spec[P any] struct {
	Name        string
	Description string // 做什么、何时用（系统提示词不再逐个介绍工具）
	// Permission 看得到、调得了这个工具所需的权限
	Permission string
	// PermissionFor 按参数细分调用所需的权限（如 query_cluster 按动作）；nil 时用 Permission。
	// 返回值必须在 Permissions 里（测试核对）
	PermissionFor func(P) string
	Permissions   []string
	Refine        func(*jsonschema.Schema)
	Handler       func(ctx context.Context, clusterID string, p P) (string, error)
}

// Tool 一个已定义好的工具
type Tool struct {
	name        string
	description string
	schemaJSON  json.RawMessage
	resolved    *jsonschema.Resolved
	permission  string
	permissions []string
	run         func(ctx context.Context, clusterID string, allow PermissionCheck, raw []byte) (string, error)
}

// Define 由参数结构体生成 schema 并定义工具；schema 生成失败是编程错误
func Define[P any](spec Spec[P]) (*Tool, error) {
	schema, err := jsonschema.For[P](nil)
	if err != nil {
		return nil, fmt.Errorf("工具 %s 的参数结构体生成 schema 失败: %w", spec.Name, err)
	}
	if spec.Refine != nil {
		spec.Refine(schema)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("工具 %s 的 schema 无效: %w", spec.Name, err)
	}
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("工具 %s 的 schema 序列化失败: %w", spec.Name, err)
	}
	permissions := spec.Permissions
	if len(permissions) == 0 {
		permissions = []string{spec.Permission}
	}

	t := &Tool{
		name:        spec.Name,
		description: spec.Description,
		schemaJSON:  schemaJSON,
		resolved:    resolved,
		permission:  spec.Permission,
		permissions: permissions,
	}
	t.run = func(ctx context.Context, clusterID string, allow PermissionCheck, raw []byte) (string, error) {
		params, err := decodeParams[P](resolved, raw)
		if err != nil {
			return "", err
		}
		need := spec.Permission
		if spec.PermissionFor != nil {
			need = spec.PermissionFor(params)
		}
		if need != PermNone && !allow(need) {
			return "", fmt.Errorf("%w：调用 %s 需要 %s", ErrPermissionDenied, spec.Name, need)
		}
		return spec.Handler(ctx, clusterID, params)
	}
	return t, nil
}

// MustDefine Define 的启动期版本：schema 有误直接 panic（编程错误，测试会先发现）
func MustDefine[P any](spec Spec[P]) *Tool {
	t, err := Define(spec)
	if err != nil {
		panic(err)
	}
	return t
}

// decodeParams 先按 schema 校验，再解码进参数结构体（未知字段拒绝）
func decodeParams[P any](resolved *jsonschema.Resolved, raw []byte) (P, error) {
	var params P
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = []byte("{}")
	}
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		return params, fmt.Errorf("%w：不是合法的 JSON: %v", ErrInvalidParams, err)
	}
	if err := resolved.Validate(instance); err != nil {
		return params, fmt.Errorf("%w：%v", ErrInvalidParams, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&params); err != nil {
		return params, fmt.Errorf("%w：%v", ErrInvalidParams, err)
	}
	return params, nil
}

// Name 工具名
func (t *Tool) Name() string { return t.name }

// Definition 发给模型的定义
func (t *Tool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: t.name, Description: t.description, Parameters: t.schemaJSON}
}

// Registry 工具注册表。启动时注册、运行中只读
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
	order []string
}

// NewRegistry 空注册表
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]*Tool)}
}

// Register 注册工具；重名是编程错误
func (r *Registry) Register(tools ...*Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range tools {
		if _, dup := r.tools[t.name]; dup {
			panic(fmt.Sprintf("工具 %s 重复注册", t.name))
		}
		r.tools[t.name] = t
		r.order = append(r.order, t.name)
	}
}

// Definitions 发给模型的清单：已注册 ∩ 调用者有权限（按注册顺序）。allow 为 nil 时一个也不给
func (r *Registry) Definitions(allow PermissionCheck) []llm.ToolDefinition {
	if allow == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]llm.ToolDefinition, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		if t.permission == PermNone || allow(t.permission) {
			defs = append(defs, t.Definition())
		}
	}
	return defs
}

// Execute 执行一次工具调用：未注册 / 参数不合法 / 权限不足都返回对应的分类错误；allow 为 nil 一律拒绝
func (r *Registry) Execute(ctx context.Context, clusterID string, allow PermissionCheck, call *llm.ToolCall) (string, error) {
	r.mu.RLock()
	t, ok := r.tools[call.Name]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("%w：%s", ErrUnknownTool, call.Name)
	}
	if allow == nil {
		return "", fmt.Errorf("%w：未提供调用者权限，拒绝调用 %s", ErrPermissionDenied, call.Name)
	}
	return t.run(ctx, clusterID, allow, []byte(call.Params))
}

// Permissions 全部工具可能要求的权限（去重排序；给网关测试核对都在权限表里）
func (r *Registry) Permissions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []string
	for _, t := range r.tools {
		for _, p := range t.permissions {
			if p != PermNone && !slices.Contains(all, p) {
				all = append(all, p)
			}
		}
	}
	slices.Sort(all)
	return all
}

// Names 已注册的工具名（按注册顺序）
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.order)
}
