package control

// CommandSet —— 某个集群的“命令副本”
// ---------------------------------------------
// - Master 端维护每个集群对应的一份命令集合（CommandSet）
// - 每次集合有变化（新增/修改/删除命令）时，版本号 RV 自增
// - Agent 通过 List+Watch 获取最新 CommandSet，
//   与本地保存的旧副本对比后执行对应命令
type CommandSet struct {
    ClusterID string    `json:"clusterID"` // 集群唯一 ID（区分不同 Agent/集群）
    RV        uint64    `json:"rv"`        // 版本号：每次修改命令集时自增，用于判断是否有更新
    Commands  []Command `json:"commands"`  // 当前集群的完整命令列表（全量快照）
}

// Command —— 单条具体命令
// ---------------------------------------------
// - 每个 Command 描述一条需要 Agent 执行的操作（如重启 Pod / 更新镜像 / 缩放副本）
// - 通过 ID 唯一标识，Idem 保证幂等，Op 指示增量操作类型
type Command struct {
    ID     string            `json:"id"`   // 命令唯一 ID（Master 生成，用于跟踪和审计）
    Type   string            `json:"type"` // 命令类型，例如 PodRestart / UpdateImage / ScaleDeployment
    Target map[string]string `json:"target"` // 命令目标（如 {"ns":"default","pod":"nginx-1234"}）
    Args   map[string]any    `json:"args"`   // 命令参数（如 {"replicas":3} 或 {"newImage":"repo/app:v1.2.4"}）
    Idem   string            `json:"idem"`  // 幂等键：确保即使命令被重复下发，也不会重复执行
    Op     string            `json:"op"`    // 操作语义：add=新增，update=更新已有命令，cancel=取消命令
}

// AckResult —— Agent 执行结果回执
// ---------------------------------------------
// - Agent 在执行完命令后，会将结果（AckResult）回报给 Master
// - Master 根据结果更新命令状态、写入审计日志，必要时做重试或清理
type AckResult struct {
    CommandID  string `json:"commandID"`
    Status     string `json:"status"`
    Message    string `json:"message,omitempty"`
    ErrorCode  string `json:"errorCode,omitempty"`
    StartedAt  string `json:"startedAt,omitempty"`
    FinishedAt string `json:"finishedAt,omitempty"`
    Attempt    int    `json:"attempt,omitempty"`
}
