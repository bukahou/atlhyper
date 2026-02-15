# 节点指标 OTel 迁移 — Phase 3 剩余工作

> 原设计文档: [Phase 3](../archive/node-metrics-phase3-master.md)
> 本文档仅描述**尚未完成**的剩余工作。

---

## 现状总结

Phase 1（基础设施部署）和 Phase 2（Agent 改造）已完成。Phase 3 原计划 4 项任务：

| 任务 | 状态 | 说明 |
|------|------|------|
| 前端 PSI 卡片简化（三窗口 → 单数字） | ✅ 已完成 | `PSICard.tsx` 已使用 `cpuSomePercent` 单一百分比 |
| 前端 TCP 卡片调整 | ✅ 已完成 | `TCPCard.tsx` 已使用正确字段 |
| style-preview mock 数据对齐真实格式 | ✅ 已完成 | mock 数据已使用 OTel 格式 |
| 下线 atlhyper-metrics DaemonSet | ❌ 未完成 | 部署文件仍存在 |

### 新发现的问题

在实际代码审查中发现以下未列入原 tracker 但需完成的工作：

| 问题 | 影响 |
|------|------|
| 节点指标组件硬编码英文 | PSICard/TCPCard 等 11 个组件中的文本未走 i18n |
| `api/metrics.ts` 死代码 | 旧指标 API，无任何组件引用 |

---

## 剩余工作

### 任务 1: i18n 国际化（11 个指标组件）

#### 问题

`app/system/metrics/components/` 下所有指标卡片组件使用**硬编码英文**，未通过 i18n 系统管理。例如：

```tsx
// PSICard.tsx 第 24 行
<h3>Pressure Stall Information</h3>
<p>% of time tasks stalled waiting for resources</p>

// TCPCard.tsx
<h3>TCP / Network Stack</h3>
```

这违反 CLAUDE.md 中「所有用户可见文本必须通过 i18n 管理」的规则。

#### 涉及组件

```
atlhyper_web/src/app/system/metrics/components/
├── PSICard.tsx                               [修改] 9+ 处硬编码文本
├── TCPCard.tsx                               [修改] 15+ 处硬编码文本（含 Softnet）
├── SystemResourcesCard.tsx                   [修改] 10+ 处硬编码文本（含 NTP）
├── VMStatCard.tsx                            [修改] 8+ 处硬编码文本
├── CPUCard.tsx                               [修改] 审查硬编码文本
├── MemoryCard.tsx                            [修改] 审查硬编码文本
├── DiskCard.tsx                              [修改] 审查硬编码文本
├── NetworkCard.tsx                           [修改] 审查硬编码文本
├── TemperatureCard.tsx                       [修改] 审查硬编码文本
├── ProcessTable.tsx                          [修改] 审查硬编码文本
└── GPUCard.tsx                               [修改] 审查硬编码文本
```

#### i18n 文件变更

```
atlhyper_web/src/
├── i18n/
│   ├── locales/
│   │   ├── zh.ts                             [修改] 新增 nodeMetrics 翻译段
│   │   └── ja.ts                             [修改] 新增 nodeMetrics 翻译段
│   └── types/
│       └── i18n.ts                           [修改] 新增 nodeMetrics 类型定义
```

#### 需要的翻译键（示例）

```typescript
nodeMetrics: {
  // PSI
  psiTitle: "压力阻塞信息",
  psiDescription: "任务因资源不足阻塞的时间占比",
  psiSome: "部分阻塞 (至少一个任务)",
  psiFull: "完全阻塞 (所有任务)",

  // TCP
  tcpTitle: "TCP / 网络栈",
  established: "已建立连接",
  timeWait: "TIME_WAIT",
  orphan: "孤儿连接",
  socketAlloc: "已分配 Socket",
  socketInUse: "使用中 Socket",
  socketsUsed: "已用 Socket",
  softnetDropped: "软中断丢包",
  softnetSqueezed: "软中断挤压",

  // System Resources
  systemTitle: "系统资源",
  conntrack: "连接跟踪",
  fileDescriptors: "文件描述符",
  entropy: "可用熵",

  // NTP
  ntpTitle: "时间同步",
  ntpOffset: "时间偏移",
  ntpSynced: "已同步",
  ntpNotSynced: "未同步",

  // VMStat
  vmstatTitle: "虚拟内存统计",
  pageFaults: "页错误/秒",
  majorPageFaults: "主页错误/秒",
  swapIn: "换入/秒",
  swapOut: "换出/秒",

  // 通用
  cpu: "CPU",
  memory: "内存",
  disk: "磁盘",
  network: "网络",
  temperature: "温度",
  processes: "进程",
  // ...
}
```

### 任务 2: 下线 atlhyper-metrics DaemonSet

#### 问题

旧的 `atlhyper-metrics` DaemonSet（v1 架构）部署配置仍然存在。OTel Collector 已完全替代其功能。

#### 文件变更

```
deploy/
├── k8s/
│   └── atlhyper-metrics.yaml                [删除] 旧 DaemonSet 部署文件
├── helm/
│   └── templates/
│       └── metrics-daemonset.yaml           [删除] 旧 Helm 模板
```

#### 注意事项

- 删除前确认已在所有集群中停止该 DaemonSet
- 如果 Helm chart 中有 `values.yaml` 引用了 metrics 相关配置，也需清理

### 任务 3: 删除旧指标 API（与 Phase 5 合并）

#### 问题

`api/metrics.ts` 是旧指标 API 客户端（调用 `/uiapi/metrics/*`），已被 `api/node-metrics.ts`（调用 `/api/v2/node-metrics/*`）完全替代。无任何组件引用。

```
atlhyper_web/src/
├── api/
│   └── metrics.ts                            [删除] 旧指标 API，被 node-metrics.ts 替代
```

此任务与「大后端小前端 Phase 5」重叠，执行一次即可。

---

## 执行顺序

```
任务 1 (i18n) 和 任务 2 (DaemonSet 下线) 可并行
  │
  ├── 任务 1: i18n
  │     ├── Step 1: i18n 类型定义 + 翻译文件
  │     ├── Step 2: 逐个组件替换硬编码文本
  │     └── 验证: npx next build
  │
  ├── 任务 2: DaemonSet 下线
  │     └── 删除部署文件
  │
  └── 任务 3: 删除 api/metrics.ts
        └── 验证: npx next build
```

---

## 文件变更总结

| 操作 | 文件 | 说明 |
|:----:|------|------|
| **修改** | `atlhyper_web/src/app/system/metrics/components/PSICard.tsx` | i18n |
| **修改** | `atlhyper_web/src/app/system/metrics/components/TCPCard.tsx` | i18n |
| **修改** | `atlhyper_web/src/app/system/metrics/components/SystemResourcesCard.tsx` | i18n |
| **修改** | `atlhyper_web/src/app/system/metrics/components/VMStatCard.tsx` | i18n |
| **修改** | `atlhyper_web/src/app/system/metrics/components/CPUCard.tsx` | i18n（审查后确定） |
| **修改** | `atlhyper_web/src/app/system/metrics/components/MemoryCard.tsx` | i18n（审查后确定） |
| **修改** | `atlhyper_web/src/app/system/metrics/components/DiskCard.tsx` | i18n（审查后确定） |
| **修改** | `atlhyper_web/src/app/system/metrics/components/NetworkCard.tsx` | i18n（审查后确定） |
| **修改** | `atlhyper_web/src/app/system/metrics/components/TemperatureCard.tsx` | i18n（审查后确定） |
| **修改** | `atlhyper_web/src/app/system/metrics/components/ProcessTable.tsx` | i18n（审查后确定） |
| **修改** | `atlhyper_web/src/app/system/metrics/components/GPUCard.tsx` | i18n（审查后确定） |
| **修改** | `atlhyper_web/src/i18n/locales/zh.ts` | +nodeMetrics 翻译段 |
| **修改** | `atlhyper_web/src/i18n/locales/ja.ts` | +nodeMetrics 翻译段 |
| **修改** | `atlhyper_web/src/i18n/types/i18n.ts` | +nodeMetrics 类型 |
| **删除** | `atlhyper_web/src/api/metrics.ts` | 旧指标 API（与 Phase 5 合并） |
| **删除** | `deploy/k8s/atlhyper-metrics.yaml` | 旧 DaemonSet |
| **删除** | `deploy/helm/templates/metrics-daemonset.yaml` | 旧 Helm 模板 |

共 ~17 文件变更（0 新增，14 修改，3 删除）。
