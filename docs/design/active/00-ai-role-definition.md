# AI 角色定义

> 状态: active | 创建: 2026-03-06
> 关联文档:
> - [03-ai-role-routing-design.md](./03-ai-role-routing-design.md) — 路由基础设施（Provider 分配/预算/上下文管理）
> - [04-ai-background-analysis-design.md](./04-ai-background-analysis-design.md) — background + analysis 功能实现设计
> - [02-ai-reports-storage-design.md](./02-ai-reports-storage-design.md) — 报告持久化（存储层设计）

## 概述

AtlHyper AI 模块定义三个工作角色，对应不同的使用场景和模型要求。
每个角色可绑定不同的 LLM Provider，由角色路由系统分发。

---

## 角色关系图

```
信号产生
  |
  v
AIOps Engine（基线/异常/事件）
  |
  |--自动--> background（快速摘要，每个事件都跑，Ollama）
  |            输出: 1-2 句话的 summary，存入 DB
  |
  |--按需--> analysis（深度调查，重要事件才跑，Gemini Pro）
  |            输出: 完整分析报告
  |            触发: 用户手动 / severity=critical 自动升级
  |
  +--交互--> chat（用户对话，随时提问，Gemini Flash）
               输出: SSE 对话流
               触发: 用户在 Chat 页面发消息
```

类比:
- **background** = 自动体检报告（每人都做，基础检查）
- **analysis** = 专科会诊报告（疑难杂症才请，全面深入）
- **chat** = 和医生直接对话（你问什么他答什么）

---

## 角色定义

### background（后台分析）

| 项目 | 说明 |
|------|------|
| **定位** | L3 持续推理 — "永不休息的值班员" |
| **驱动方** | 系统自动（无用户参与） |
| **触发条件** | 新事件创建 / 事件状态变化 / 定时巡检 |
| **交互模式** | 单轮 LLM 调用，异步后台执行 |
| **输入** | AIOps Engine 提供的事件数据（实体/时间线/历史） |
| **输出** | 结构化 JSON（摘要/根因/建议），持久化到 DB |
| **Tool Calling** | 不需要（数据由 Engine 直接提供） |
| **频率** | 高（每个 incident 至少一次） |
| **模型倾向** | 低成本、可用即可（如 Ollama 本地、Gemini Flash） |

### chat（交互对话）

| 项目 | 说明 |
|------|------|
| **定位** | 用户引导的交互式调查 |
| **驱动方** | 用户提问 |
| **触发条件** | 用户在 Chat 页面发送消息 |
| **交互模式** | SSE 流式多轮对话 |
| **输入** | 用户消息 + 对话历史 |
| **输出** | 实时文本流 + Tool 调用过程展示 |
| **Tool Calling** | 最多 5 轮 x 5 并行，用户可见每一步 |
| **频率** | 中（用户按需） |
| **模型倾向** | 快速响应 + 强工具调用能力（如 Gemini Flash、GPT-4o-mini） |

### analysis（深度分析）

| 项目 | 说明 |
|------|------|
| **定位** | L4 专家级诊断 — "请来的专科医生" |
| **驱动方** | 用户触发或系统自动升级 |
| **触发条件** | 用户点击"深度分析" / severity=critical 自动升级 |
| **交互模式** | 一次触发，AI 自主调查，完成后通知 |
| **输入** | 事件 ID（AI 自主决定需要查什么数据） |
| **输出** | 结构化分析报告（根因链/证据/建议/历史对比） |
| **Tool Calling** | 多轮自主调用（和 chat 共享 Tool，但无用户参与） |
| **频率** | 低（仅重要/复杂事件） |
| **模型倾向** | 强推理 + 大上下文（如 Gemini Pro、Claude、GPT-4o） |

> **角色与 Provider 不硬绑定**：任何角色可分配给任何 Provider，用户在 Web UI 随时切换。
> 上述"模型倾向"仅为选型参考，不构成约束。上下文管理、Token 裁剪等自动适配当前 Provider 的模型能力。
> 详见 [03-ai-role-routing-design.md](./03-ai-role-routing-design.md)。

---

## chat vs analysis 关键区别

| | chat | analysis |
|--|------|----------|
| **谁在驱动调查** | 用户引导方向 | AI 自主决策 |
| **用户体验** | 实时看到每一步 | 触发后等报告 |
| **输出形态** | 对话流 | 结构化报告 |
| **执行方式** | 同步（SSE 推送） | 异步（后台任务） |
| **适合场景** | 探索性问题 | 已知事件的系统化调查 |

---

## 后端实现现状

| 角色 | 代码路径 | 状态 |
|------|---------|------|
| **chat** | `ai/chat.go` -> `chatLoop` | 已完善（多轮 Tool Calling + SSE + 持久化） |
| **background** | `aiops/ai/enhancer.go` -> `Summarize` | 基本可用（单轮 LLM，手动触发），需改为自动触发 |
| **analysis** | -- | 未实现（无代码路径） |

详细实现设计见 [04-ai-background-analysis-design.md](./04-ai-background-analysis-design.md)。

---

## 角色常量

```go
// ai/role.go

const (
    RoleBackground = "background"  // 后台分析
    RoleChat       = "chat"        // 交互对话
    RoleAnalysis   = "analysis"    // 深度分析
)

var ValidRoles = []string{RoleBackground, RoleChat, RoleAnalysis}
```
