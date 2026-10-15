# AIOps 探针噪音治理 — 设计文档

> 状态：设计已定（2026-10-15 用户裁定「按推荐来」），代码已完成，待 dev 验收
> 任务追踪：`docs/tasks/active/tracker.md`「AIOps 探针噪音治理」
> 前置：[aiops-incident-evidence-design.md](aiops-incident-evidence-design.md)（本设计复用其持久化 event 与触发信号）

## 背景

prod 换上新版本后，2026-10-15 13:59–16:50 共产生 12 条事件，每条都自动触发了一次后台 AI 分析。逐条核对时间线里的 `anomaly_detected`，触发信号**全部是同一类**：

```
K8s 事件 Unhealthy: Readiness probe failed: Get "http://<pod-ip>:<port>/<path>":
  context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

（其中 2 条是 Liveness probe failed，其余是 Readiness。）

| 事实 | 观测 |
|------|------|
| 涉及的应用 | 只有两个应用的 API Pod，跨两个节点，多次同一秒一起超时 |
| 同节点的其他应用 | 同样是 1s 探针，没有失败 |
| Pod 是否真的受影响 | 0 次重启；每次只失败 1 次，没到 `failureThreshold: 3`，Pod 没有被摘流量 |
| 事件结局 | 风险峰值都是 0.48（没到 incident 的 0.5），约 6 分钟后自动转为 stable |

结论：这 12 条都是**偶发、自愈、对用户无影响的噪音**，而 atlhyper 把每一条都当成了事件。探针为什么会超时，属于被监控应用自己的事，不在本设计范围内（见「非目标」）。

### 一次探针失败怎么变成一条事件（已对照代码确认）

```
kubelet 探针失败 1 次（还没到 failureThreshold，Pod 仍 Ready）
  └─ K8s 发出 Warning 事件 Unhealthy（每失败一次就发一条）
       └─ cluster.Event.IsCritical() 把 Unhealthy 列为 critical
            └─ baseline.extractEventAnomalies：最近 5 分钟内有 critical 事件 → critical_event，打 0.85 分
                 └─ 异常因此持续 5 分钟 > 「风险 > 0.2 持续 2 分钟 → warning」的门槛
                      └─ statemachine → OnWarningCreated → 建事件 + incident_created → 后台 AI 分析
```

问题在第 3、4 步：**探针失败只是过程，不是症状**。同一 Pod 真正有影响的两种情况，现有代码已经从 Pod 的**实时状态**检测到了（`baseline/extractor.go` 的 `classifyContainerAnomaly`）：

| 真实症状 | 已有信号 |
|----------|----------|
| 就绪探针连续失败 → Pod 被摘流量 | 容器 `running` 但 `Ready=false` → `container_anomaly: NotReady` |
| 存活探针连续失败 → 容器被杀重启 | 10 分钟内刚终止过 → `container_anomaly: RecentCrash` / `OOMKilled`；另有 `restart_count` 统计异常 |

所以 `Unhealthy` 事件在打分上**唯一多出来的作用**，就是把还没造成影响的瞬时失败也算成异常。

## 目标

- 探针偶发失败、Pod 仍然 Ready 时，**不产生异常，也不建事件**
- 探针失败真的造成影响（NotReady / 重启）时，**照常检测，不能漏**
- `Unhealthy` 事件的原文**仍然留作证据**：持久化、AI 分析上下文、对话工具 `get_event_history` 都能看到

**非目标**：
- 被监控应用的探针设计（例如 DB ping 写在 liveness 里、1s 超时）——属于对方仓库，已转交对方会话，要定案需走对线
- 自动触发 AI 分析的级别：用户在 AI 设置页把「自动触发最低严重级别」调到「中」，不改代码（2026-10-15 裁定）
- 其他 critical 事件原因（`BackOff` / `Failed` / `FailedMount` / `FailedScheduling` 等）：它们出现本身就意味着有实际故障，且本次没有一条噪音来自它们。没有证据就不扩大改动范围
- 前端「持续时间」口径、`/events?source=history` 忽略 namespace —— 仍为独立待办

## 决策记录（2026-10-15 用户裁定）

### 决策 1：`Unhealthy` 事件还算不算异常

**背景**：业界告警的原则是看症状、不看过程（Google SRE「alert on symptoms」）。探针失败是过程，摘流量和重启才是症状，而症状已有独立信号。

| 方案 | 影响 |
|------|------|
| A. 维持现状 | 每天几十条噪音，AI 每日调用额度会被它们先用光 |
| **B. 只当证据，不打分** ✅ | 瞬时失败不再成为事件；真有影响时由 `NotReady` / `RecentCrash` 照常触发，证据里仍能看到 `Unhealthy` 原文 |
| C. 降低分数（0.85 → 0.3 之类） | 只能减少一部分；以后风险的计算方式一改，分数又得重调。治标 |
| D. 累计到 `failureThreshold` 次才算 | agent 拿不到每个探针的门槛，要改采集；而累计到门槛时 Pod 已 NotReady，B 本来就能检测到 —— 结果相同，成本更高 |

**裁决**：B。
**已接受的风险**：Ready 状态的极短闪断，如果恰好落在两次快照之间（快照间隔为秒级），会漏检。这种时长造成的影响本身很小。

### 决策 2：范围只限 `Unhealthy`

**裁决**：只改 `Unhealthy`，其他 critical 原因保持不变（理由见「非目标」）。

### 决策 3：不在代码里改 AI 自动触发逻辑

**裁决**：用户自行把「自动触发最低严重级别」调到「中」，作为额外一层保险。B 做完后噪音从源头消失，不需要再加一套重复的代码开关。

## 功能：`Unhealthy` 降为证据型事件

### 用户故事

作为平台维护者，我希望只有在 Pod 真被摘流量或被重启时才看到事件。偶发的探针超时应该留在事件记录里备查，但不要每次都生成一条事件，并消耗一次 AI 分析。

### 实现位置（工程决策：不改共用模型）

`cluster.Event.IsCritical()` 定义在 `model_v3`（agent 和 master 共用的领域模型）中。它的语义是「这条 K8s 事件本身是否严重」，对 `Unhealthy` 来说并没有错：一条探针失败事件确实是 Warning 级别的严重事件。

要改的是 **AIOps 怎么使用它**：哪些严重事件可以单独拿来打分。因此改动只放在 AIOps 的提取层：

```go
// atlhyper_master_v2/aiops/baseline/extractor.go

// evidenceOnlyReasons 只当证据、不单独打分的 critical 事件原因。
// Unhealthy = kubelet 每失败一次探针就发一条，此时 Pod 往往仍然 Ready（还没到 failureThreshold）。
// 真正的影响（摘流量 / 重启）由 classifyContainerAnomaly 从 Pod 实时状态判定（NotReady / RecentCrash）。
var evidenceOnlyReasons = map[string]bool{
    "Unhealthy": true,
}
```

`extractEventAnomalies` 在 `IsCritical()` 判断之后加一句：`if evidenceOnlyReasons[ev.Reason] { continue }`。

核对过的影响面（`git grep IsCritical / GetSeverity`）：
- `IsCritical()` 在生产代码中**只有** `extractEventAnomalies` 一个调用方，另一个是 `GetSeverity()`
- `GetSeverity()` 没有任何调用方 ⇒ 不改 `model_v3` 不会影响其他功能；若改了 `model_v3`，反而会悄悄改变它的语义

### 证据通路不受影响（核对点）

`Unhealthy` 事件的另外三条通路都**不经过** `extractEventAnomalies`，因此不受影响：

| 通路 | 位置 | 依据 |
|------|------|------|
| 持久化 | `service/sync` EventPersist → `cluster_events` 表 | 按 Warning 类型持久化，不看 IsCritical |
| AI 分析上下文 | `aiops/enricher/event_context.go` `collectWindowEvents` | 直接读 `cluster_events` |
| 对话工具 | `ai/tool_event_history.go` | 直接读 `cluster_events` |

另外：如果同一个 Pod 因为 `NotReady` 或 `RecentCrash` 进入了事件，AI 分析上下文里照样能看到这段时间的 `Unhealthy` 原文。证据链是完整的。

### 测试（TDD，先红后绿）

| 测试 | 文件 | 断言 |
|------|------|------|
| `TestExtractEventAnomalies_UnhealthyIsEvidenceOnly` [新增] | `baseline/extractor_test.go` | Pod 仍 Ready，5 分钟内有 `Unhealthy`（Readiness 和 Liveness 各一例）→ **不**产生 `critical_event` |
| `TestExtractEventAnomalies_UnhealthyDoesNotMaskOtherReasons` [新增] | 同上 | 同一个 Pod 既有较新的 `Unhealthy`，又有较早的 `BackOff` → 仍产生 `critical_event`，且 Detail 取 `BackOff`（证据型事件不参与「取最近一条」的比较） |
| `TestExtractDeterministic_NotReady` [已有，复用] | 同上 | 容器 `running` 且 `Ready=false` → 产生 `container_anomaly: NotReady`（0.60）—— 证明真实症状没有被一起去掉，必须保持绿 |
| `TestExtractEventAnomalies_DetailUsesLatestEvent` [修改] | `baseline/signal_detail_test.go` | 原来用 `Unhealthy` 当「较新的事件」，改成另一个仍参与打分的原因（如 `Failed`），测试意图不变 |

现有的 `CriticalPodEvent` / `OldEvent` / `NonPodEvent` / `DeduplicatePerPod` 用的是 `BackOff` 等原因，应保持全绿。

## 实施阶段

单阶段即可完成闭环：

1. 设计文档 + tracker ✅（本文件）
2. 写测试 → 红灯确认
3. 实现（一个 map 加一行判断）→ 绿灯；`go test ./atlhyper_master_v2/aiops/...` 全绿
4. 合 dev → CI → dev 部署（push 需当次批准）
5. **验收**：dev 上观察 24 小时
   - 事件数量：对比 dev 部署前后的同等时长（预期大幅下降；akasha / melete 的探针抖动不再成为事件）
   - 回归：真实 NotReady / 重启的 Pod 仍然会进入事件（可在 dev 上找一个自然发生的，或由用户决定是否人为制造）
6. 开 dev → main 的 PR，用户手动合并（本次不涉及表结构变更，**不需要清库**）

## 文件变更清单

```
atlhyper_master_v2/aiops/baseline/
├── extractor.go            [修改] evidenceOnlyReasons + extractEventAnomalies 跳过
├── extractor_test.go       [修改] 新增 2 个测试
└── signal_detail_test.go   [修改] DetailUsesLatestEvent 换掉 Unhealthy
docs/
├── design/active/aiops-probe-noise-design.md   [新增] 本文件
└── tasks/active/tracker.md                     [修改] 新增任务组
```

不改：`model_v3/`（共用模型）、状态机阈值、AI 自动触发逻辑、前端。
