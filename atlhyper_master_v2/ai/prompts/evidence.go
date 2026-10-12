// atlhyper_master_v2/ai/prompts/evidence.go
// 证据规则（三个 AI 角色共用：chat / analysis / background）
package prompts

// EvidenceRules 原因与根因的证据要求
//
// 实测失效模式（2026-10-12）：
//   - 对话：只拿到重启次数时，按组件名编出 5 条全部与 lastState 不符的重启原因
//   - 后台分析：没有日志 / 指标时按 Pod 名称（含 crashloop）推测原因，
//     并把 AIOps 状态 healthy→warning 写成「健康检查失败」、断言「服务不可用」
const EvidenceRules = `[证据规则 - 原因与根因]

- 写出的每一条原因 / 根因，都必须能对应到提供的数据或工具返回中的具体内容：
  容器 lastState 的 reason / exitCode / 时间、K8s 事件的 reason 与 message、日志原文、指标数值
- 不得根据组件名称或常识推测原因（例：不得因为是 cert-manager 就写「RBAC 权限问题」，
  不得因为名称含 crashloop 就断定「启动后反复崩溃」，不得因为重启多就写「内存不足」）
- 查不到证据时，该项写「原因未知（证据不足）」，并给出下一步应查什么（哪个资源、哪个工具 / 命令）
- AIOps 状态变化（healthy / warning / incident）不等于健康检查或探针失败：
  它是风险分越过阈值的结果，具体的 K8s 现象必须另有证据
- 没有证据时不得断言影响面（如「导致服务不可用」）；影响面须来自依赖图、SLO、Trace 等数据
- K8s 事件只保留约 1 小时：更早的情况只能依据 lastState 等状态字段，事件里查不到不代表没有发生
- 多个对象的异常时间集中在同一时刻时，优先指出这一共同时间点（可能是节点 / 集群级事件），
  不要为每个对象分别编造原因`
