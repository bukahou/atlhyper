# AI background + analysis 功能实现设计

> 状态: active (讨论中) | 创建: 2026-03-06
> 关联文档:
> - [ai-role-definition-design.md](./ai-role-definition-design.md) — 角色定义（是什么）
> - [ai-role-routing-design.md](./ai-role-routing-design.md) — 路由基础设施（Provider 分配/预算/上下文管理）
> - [ai-reports-persistence-design.md](./ai-reports-persistence-design.md) — 报告持久化（存储层设计）

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

### 需要实现的内容

#### 1. 事件创建时自动触发摘要

TODO: 待讨论
- AIOps Engine 创建 Incident 后异步调用 Enhancer
- 依赖注入方式
- 错误处理策略
- 结果写入方式（aiops_incidents.summary 字段已存在）

#### 2. 事件状态变化时更新摘要

TODO: 待讨论
- 哪些状态变化需要重新分析
- 缓存失效策略

#### 3. 定时巡检摘要（后续扩展）

TODO: 待讨论
- 巡检频率和内容
- 报告存储模型
- 前端展示位置

### 改造范围评估

TODO: 待确认

---

## 二、analysis（深度分析）实现设计

### 核心问题

analysis 是全新功能，需要回答以下设计问题:

#### 1. 触发方式

TODO: 待讨论
- 用户手动触发（事件详情页"深度分析"按钮）
- 系统自动升级（severity=critical 时自动触发）
- 两者都支持？

#### 2. 执行模型

TODO: 待讨论
- 异步任务系统设计（触发 -> 执行 -> 完成通知）
- 与 chat 的 Tool Calling 循环如何复用
- 执行超时策略

#### 3. 输出报告模板

TODO: 待讨论
- 报告包含哪些章节
- 存储模型（新表 or 复用现有表）
- 前端展示方式

#### 4. 自主调查流程

TODO: 待讨论
- AI 如何决定需要查什么数据
- 调查步骤的 prompt 设计
- 与 chat 的 Tool 定义是否共享

#### 5. 前置依赖

| 依赖 | 说明 | 现状 |
|------|------|------|
| 自主 Tool Calling 循环 | 类似 chatLoop 但无 SSE，后台执行 | 可复用 chat 的 Tool 基础设施 |
| 异步任务系统 | 触发 -> 执行 -> 完成通知 | 不存在，需设计 |
| 报告模板 + 存储 | 结构化输出定义 + DB 模型 | 不存在，需设计 |
| 报告展示页 | 前端渲染分析报告 | 不存在，需设计 |
| APM + Logs Tool | analysis 需要查询 Trace/Log 数据 | AIOps OTel 融合进行中 |

---

## 实施优先级

| 优先级 | 内容 | 改造量 |
|--------|------|--------|
| **P1** | background: 事件自动摘要（Engine 创建事件时触发） | 小 |
| **P1** | background: 状态变化时更新摘要 | 小 |
| **P2** | background: 定时巡检报告 | 中 |
| **P3** | analysis: 完整设计 + 实现 | 大 |
