package model

// ListOptions 列表查询选项
//
// 用于 Repository 和 Service 层的资源列表查询
type ListOptions struct {
	// LabelSelector 标签选择器
	// 格式: "key=value,key2=value2" 或 "key in (v1,v2)"
	// 示例: "app=nginx,env=prod"
	LabelSelector string

	// FieldSelector 字段选择器
	// 格式: "field.path=value"
	// 示例: "status.phase=Running", "spec.nodeName=node1"
	FieldSelector string

	// Limit 限制返回数量
	// 0 表示不限制
	Limit int64

	// Namespace 命名空间过滤
	// 空表示所有命名空间
	Namespace string
}

// LogOptions 日志查询选项
//
// 用于获取 Pod 日志
type LogOptions struct {
	// Container 容器名称
	// Pod 有多个容器时必须指定
	Container string

	// TailLines 返回最后 N 行
	// 0 表示返回全部
	TailLines int64

	// SinceSeconds 返回最近 N 秒的日志
	// 0 表示不限制时间
	SinceSeconds int64

	// Timestamps 是否包含时间戳
	Timestamps bool

	// Follow 是否跟踪日志 (流式)
	// 目前不支持
	Follow bool

	// Previous 是否获取之前容器的日志
	// 用于查看重启前的日志
	Previous bool
}

// ExecOptions 执行命令选项
//
// 用于在 Pod 容器中执行命令
type ExecOptions struct {
	// Container 容器名称
	Container string

	// Command 要执行的命令
	// 示例: []string{"sh", "-c", "ls -la"}
	Command []string

	// TTY 是否分配 TTY
	TTY bool

	// Stdin 是否使用 stdin
	Stdin bool
}

// DeleteOptions 删除选项
type DeleteOptions struct {
	// GracePeriodSeconds 优雅终止时间 (秒)
	// nil 使用默认值，0 表示立即删除
	GracePeriodSeconds *int64

	// Force 是否强制删除
	// 设为 true 时会设置 GracePeriodSeconds=0
	Force bool

	// PropagationPolicy 级联删除策略
	// Orphan: 不删除依赖资源
	// Background: 后台删除依赖资源
	// Foreground: 前台删除依赖资源
	PropagationPolicy string
}

// ScaleOptions 扩缩容选项
type ScaleOptions struct {
	// Replicas 目标副本数
	Replicas int32
}

// PatchOptions 补丁选项
type PatchOptions struct {
	// PatchType 补丁类型
	// strategic: Strategic Merge Patch (K8s 默认)
	// merge: JSON Merge Patch
	// json: JSON Patch
	PatchType string

	// PatchData 补丁内容
	PatchData []byte
}
