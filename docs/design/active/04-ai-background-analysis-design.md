# AI background + analysis 功能实现设计

> 状态: active | 创建: 2026-03-06
> 关联文档:
> - [00-ai-role-definition.md](./00-ai-role-definition.md) — 角色定义（是什么）
> - [03-ai-role-routing-design.md](./03-ai-role-routing-design.md) — 路由基础设施（Provider 分配/预算/上下文管理）
> - [02-ai-reports-storage-design.md](./02-ai-reports-storage-design.md) — 报告持久化（存储层设计）

## 概述

chat 角色已完善，本文档设计 background 和 analysis 两个缺失角色的后端实现。

---

## 一、background（后台分析）实现设计

### 现状

- `aiops/ai/enhancer.go` 的 `Summarize()` 已有完整实现:
  - 构建事件上下文（实体/时间线/历史相似事件）
  - Token 预估 + 逐步截断
  - 单轮 LLM 调用
  - 结构化 JSON 解析
  - Rate Limit (60s 冷却) + 结果缓存
- 但触发方式是**手动**（前端事件详情页点击按钮）
- AIOps Engine 不持有 Enhancer 引用

### 行为链

```
触发 ─────────────────────────────────────────────────
  谁触发？  AIOps Engine（StateMachine 状态变迁时）
  何时触发？
    1. 新事件创建（Healthy → Warning）
    2. 事件升级（Warning → Incident）
    3. 定时巡检（Scheduler，无事件关联）

  Engine 通过事件总线发布事件，Enhancer 订阅（最大程度解耦）

路由 ─────────────────────────────────────────────────
  loadAIConfigForRole("background")
    → 查 ai_providers 谁持有 "background" 角色
    → 找到 → 检查预算 → 返回 Provider 配置
    → 找不到 → 退回全局 ai_active_config.provider_id

执行 ─────────────────────────────────────────────────
  优先级队列（单 worker）
    → 从队列取出风险分数最高的事件
    → Enhancer.Summarize(incident)
      → 构建 Prompt（实体/时间线/历史相似事件）
      → Prompt 按 context_window 截断
      → 单轮 LLM 调用（无 Tool Calling）
      → 解析结构化 JSON 响应
    → 完成后取下一个

产出 ─────────────────────────────────────────────────
  写入 ai_reports（完整报告：摘要+根因+建议+相似事件）
  同步更新 aiops_incidents.summary（列表页快速展示）
  记录 Provider/Model/Token/耗时（成本追踪）
```

### 设计决策

#### 1. Engine ↔ Enhancer 解耦：事件总线

Engine 和 Enhancer 通过事件总线通信，互不直接引用。

```
Engine（发布者）                    Enhancer（订阅者）
  │                                  │
  ├── 创建事件                        │
  │   └── bus.Publish(IncidentEvent) │
  │                                  ├── bus.Subscribe(IncidentEvent)
  │                                  │   └── 入队优先级队列
  │                                  │
  ├── 状态变迁                        │
  │   └── bus.Publish(StateChanged)  │
  │                                  ├── bus.Subscribe(StateChanged)
  │                                  │   └── 去重后入队
  │                                  │
  └── (Engine 不关心 LLM 结果)       └── (Enhancer 独立写 DB)
```

事件总线可复用现有的内存 MQ（`mq/`），或在 Engine 内新增一个简单的 channel-based bus。
Engine 只需 `Publish`，不持有任何 AI 模块引用。

#### 2. 异步执行：优先级队列 + 单 worker

不无限并发触发 LLM，而是用单 worker 逐个处理，优先分析高风险事件。

```go
// aiops/ai/queue.go

type SummaryTask struct {
    IncidentID string
    RiskScore  float64  // 优先级依据
    Trigger    string   // "incident_created" / "state_changed"
    EnqueuedAt time.Time
}

// 优先级队列（按 RiskScore 降序）
// 单 worker goroutine 消费：
//   1. 取出 RiskScore 最高的任务
//   2. 执行 Enhancer.Summarize()
//   3. 写入 ai_reports + incidents.summary
//   4. 回到 1
```

**为什么单 worker**：
- LLM 调用本身就是瓶颈（Ollama 一次只能处理一个请求）
- 避免并发写同一事件的 summary 产生竞争
- 简单可靠，不需要复杂的并发控制

**队列上限**：最多保留 50 个待处理任务，超出丢弃最低风险的。

#### 3. 失败策略：重试一次 + 熔断

LLM 调用和事件创建完全独立。事件由 AIOps Engine 自主计算产生，LLM 失败不影响事件。

```
第 1 次调用失败（超时/错误）
  → 等待 5 秒，重试 1 次
  → 第 2 次仍失败
    → 记录失败原因到日志
    → incidents.summary 写入 "AI 分析暂时不可用"
    → consecutiveFailures++

熔断机制:
  consecutiveFailures >= 3
    → 暂停 LLM 调用 5 分钟
    → 5 分钟后尝试恢复（下一个任务作为探针）
    → 探针成功 → 重置计数，恢复正常
    → 探针失败 → 暂停时间翻倍（5min → 10min → 20min，上限 1 小时）

  任何一次成功调用 → consecutiveFailures = 0，暂停时间重置
```

#### 4. 去重：同一事件只保留最新触发

同一事件在短时间内多次状态变化（Warning → Incident），只分析最终状态。

```
去重规则:
  1. 新任务入队时，检查队列中是否已有相同 IncidentID 的任务
     → 有 → 替换（用新的 RiskScore 和 Trigger 覆盖旧的）
     → 无 → 正常入队

  2. 正在执行中的任务不可替换（让它跑完）
     → 但标记 needRerun = true
     → 当前任务完成后，检查 needRerun
       → true → 用最新状态重新入队一次
       → false → 结束

  3. 刚完成分析的事件，30 秒内再次触发 → 忽略
     → 维护 recentlyAnalyzed map[incidentID]time.Time
     → 入队前检查：如果 30 秒内刚分析过，跳过
```

示例时间线：

```
T=0s   事件 A 创建 (Warning, risk=0.3)  → 入队 [A:0.3]
T=5s   事件 B 创建 (Warning, risk=0.7)  → 入队 [B:0.7, A:0.3]（B 优先）
T=8s   事件 A 升级 (Incident, risk=0.6) → 替换 [B:0.7, A:0.6]
T=10s  Worker 取出 B，开始分析
T=15s  事件 A 再次升级 (risk=0.8)       → 替换 [A:0.8]
T=25s  Worker 完成 B，取出 A(0.8)，开始分析
T=40s  Worker 完成 A
T=45s  事件 A 状态变化                   → 30 秒冷却内，跳过
T=75s  事件 A 状态变化                   → 冷却结束，允许入队
```

### 需要实现的内容

#### 1. 事件总线集成

- Engine 在 `OnSnapshot` 中状态变迁时发布事件
- Enhancer 订阅事件，入队优先级队列
- Engine 和 Enhancer 零耦合

#### 2. 优先级队列 + 单 Worker

- `aiops/ai/queue.go` — SummaryTask + 优先级队列
- 单 goroutine 消费，按 RiskScore 降序处理
- 队列上限 50，超出丢弃低风险任务

#### 3. 去重逻辑

- 入队去重：相同 IncidentID 替换
- 执行中标记 needRerun
- 完成后 30 秒冷却

#### 4. 熔断器

- 连续失败 ≥ 3 次暂停（5min → 10min → 20min，上限 1h）
- 成功即重置

#### 5. 定时巡检摘要（后续扩展）

- 巡检频率和内容 — 待讨论
- 报告存储模型
- 前端展示位置

---

## 二、analysis（深度分析）实现设计

### 行为链

```
触发 ─────────────────────────────────────────────────
  谁触发？  系统自动（高危事件触发，不需要人工触发）
  何时触发？
    1. severity=critical 的事件自动触发
    2. 人工调查需求走 chat 角色，不走 analysis

  与 background 共享优先级队列，但 analysis 任务优先级更高。

路由 ─────────────────────────────────────────────────
  loadAIConfigForRole("analysis")
    → 同 background 的路由逻辑
    → 倾向强推理模型（Gemini Pro / Claude）

执行 ─────────────────────────────────────────────────
  复用 chat 的 Tool Calling 循环（chatLoop 模式）
    → 输入: 高危事件信息 + background 已有摘要 + system prompt
    → 每轮:
      1. 发送消息（附带剩余轮次提醒）
      2. AI 返回 Tool Calls → 执行（支持并行多指令）→ 结果喂回
      3. AI 判断是否需要继续调查
         → 需要 → 继续下一轮
         → 不需要 → 返回 {"continue": false}，停止循环
      4. 记录本轮: 思考过程 + 调用指令 + 查询结果
    → 最大 8 轮，到达上限强制输出最终报告
    → 无 SSE 推送（后台静默执行）

产出 ─────────────────────────────────────────────────
  写入 ai_reports:
    - 最终报告（摘要 + 根因 + 建议）
    - 调查过程（每轮的思考/指令/结果）
    - 证据链（指标/日志/事件引用）
    - Provider/Model/Token/耗时
  事件打上"已深度分析"标签
  事件详情页可关联查看调查过程和报告
```

### 设计决策

#### 1. 执行模型：与 background 共享优先级队列

analysis 和 background 共用同一个优先级队列和单 worker。analysis 任务的优先级始终高于 background。

```
队列内容示例:
  [analysis:事件A(critical), background:事件B(0.7), background:事件C(0.3)]
  → Worker 先处理 analysis:事件A
  → 完成后处理 background:事件B
  → ...
```

不需要独立的异步任务系统——analysis 只是队列中的另一种任务类型，执行逻辑不同而已。

```go
type SummaryTask struct {
    IncidentID string
    RiskScore  float64
    Trigger    string    // "incident_created" / "state_changed" / "auto_escalation"
    Role       string    // "background" / "analysis"
    EnqueuedAt time.Time
}

// Worker 根据 Role 选择执行路径:
//   background → Enhancer.Summarize()（单轮 LLM）
//   analysis   → AnalysisLoop()（多轮 Tool Calling）
```

#### 2. Tool Calling 循环：复用 chat 的 chatLoop

复用 chat 已有的 Tool Calling 基础设施（Tool 定义、Tool 执行、消息构建），但以非交互模式运行：

```
chatLoop（chat 模式）             analysisLoop（analysis 模式）
  用户消息驱动                      系统自动驱动
  SSE 实时推送                      无推送，后台静默
  用户可中断/追问                    无用户参与
  对话历史保留在 ai_messages         调查过程保留在 ai_reports
  无轮次限制（用户控制）              最大 8 轮
```

每轮的消息格式：

```
[System Prompt]
你是 AtlHyper 深度分析引擎。正在调查一个高危事件。
当前剩余调查轮次: {remaining}/8
...（详细 prompt 待设计）

[Round 1 — AI 思考]
根据事件信息，我需要先查看受影响 Pod 的状态...
→ Tool Call: query_cluster("describe pod geass-gateway-xxx -n geass")

[Round 1 — Tool Result]
Name: geass-gateway-xxx
Status: CrashLoopBackOff
...

[Round 2 — AI 思考]
Pod 在 CrashLoopBackOff，需要查看日志确认崩溃原因...
→ Tool Call: query_cluster("logs geass-gateway-xxx -n geass --tail=100")

...

[Round N — AI 判断]
信息已足够，不需要继续调查。
→ 返回 {"continue": false, "report": {...}}
```

#### 3. 每轮记录：调查过程持久化

每完成一轮 Tool Calling，立即记录到 `ai_reports.investigation_steps`（JSON 数组追加）：

```json
[
  {
    "round": 1,
    "thinking": "根据事件信息，需要查看受影响 Pod 的状态",
    "tool_calls": [
      {
        "tool": "query_cluster",
        "params": "describe pod geass-gateway-xxx -n geass",
        "result_summary": "Pod Status: CrashLoopBackOff, Restarts: 15"
      }
    ]
  },
  {
    "round": 2,
    "thinking": "Pod 在 CrashLoopBackOff，需要查看日志",
    "tool_calls": [
      {
        "tool": "query_cluster",
        "params": "logs geass-gateway-xxx -n geass --tail=100",
        "result_summary": "java.lang.OutOfMemoryError: Java heap space"
      }
    ]
  }
]
```

**每轮都写 DB**，而非全部完成后一次写入。好处：
- 即使中途崩溃/超时，已有的调查步骤不丢失
- 前端可以在分析进行中查看已完成的步骤

#### 4. 失败策略：与 background 共享熔断器

同一个熔断器管理 background 和 analysis：
- 重试 1 次 + 连续 ≥3 次失败暂停
- 暂停时 analysis 角色标记为"已停用"
- 用户可在 Web UI 手动重新激活

#### 5. 前端关联：事件标签 + 报告链接

```
事件详情页:
  ┌──────────────────────────────────────────┐
  │ 事件 #INC-20260307-001                    │
  │ 状态: Incident  严重度: Critical          │
  │ 标签: [已深度分析 ✓]                       │
  │                                          │
  │ ── AI 分析记录 ──                         │
  │ [深度] analysis 报告    03-07 11:00       │
  │   调查: 5 轮 | Gemini Pro | 15K tokens   │
  │   根因: OOM (Java heap) → Pod 重启循环    │
  │   [查看完整报告 →]  [查看调查过程 →]       │
  │                                          │
  │ [摘要] background 摘要   03-07 09:00      │
  │   geass-gateway Pod 异常重启...           │
  └──────────────────────────────────────────┘
```

### System Prompt 设计方向

> 详细 prompt 内容待后续设计，以下为框架。

```
角色: 你是 AtlHyper 深度分析引擎，负责对高危事件进行系统化调查。

上下文:
- 事件信息: {incident_summary}
- 已有 background 分析: {background_report}
- 受影响实体: {entities}
- 当前剩余调查轮次: {remaining}/8

调查规则:
1. 基于已有 background 分析，做更深入的调查，不要重复已知信息
2. 每轮你可以调用 Tool 查询集群数据（支持并行多指令）
3. 每轮结束后判断是否需要继续，不需要则返回 {"continue": false}
4. 到达最大轮次时，必须基于现有信息输出最终报告

输出格式:
- 继续调查: 调用 Tool
- 结束调查: {"continue": false, "report": {summary, rootCause, recommendations, confidence}}
```

### 前置依赖

| 依赖 | 说明 | 现状 |
|------|------|------|
| 角色路由 | loadAIConfigForRole("analysis") | Phase 3 实现 |
| 报告存储 | ai_reports 表 | Phase 5 实现 |
| 优先级队列 | 与 background 共享 | Phase 6 实现 |
| chat Tool 基础设施 | Tool 定义 + 执行 + 消息构建 | 已有，可复用 |
| 事件标签机制 | 事件打上"已深度分析"标签 | 需新增 |

---

## 实施优先级

| 优先级 | 内容 | 改造量 | 前置条件 |
|--------|------|--------|----------|
| **P1** | background: 事件总线 + 优先级队列 + 去重 + 熔断 | 中 | Phase 3 角色路由 + Phase 5 报告存储 |
| **P1** | background: Enhancer 产出写入 ai_reports | 小 | 同上 |
| **P2** | analysis: Tool Calling 循环 + 每轮记录 + 事件标签 | 大 | Phase 6 background 队列 |
| **P3** | analysis: system prompt 设计 + 调优 | 中 | P2 |
| **P4** | background: 定时巡检报告 | 中 | P1 + 巡检调度器 |
