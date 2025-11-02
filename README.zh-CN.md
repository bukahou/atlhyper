## 🧠 项目名称：AtlHyper

### 📌 项目定位

**AtlHyper** 是一个面向 Kubernetes 集群的轻量级可观测与控制平台。它能够实时监控 Node、Pod、Deployment 等资源，执行异常检测、告警通知、AI 智能诊断，并支持从 Web 界面直接进行集群操作。系统采用前后端分离架构，非常适合中小型集群、边缘节点环境以及开发测试场景。

本项目采用 **Master-Agent 模型**。Agent 部署在每个 Kubernetes 集群内部，用于采集与上报数据；Master 则运行在外部（推荐使用 Docker Compose 部署），用于集中管理、控制与协调。Master 通过 HTTP 与各 Agent 通信，并与 AI 智能诊断服务（AI Service）及外部监控数据适配模块（Adapter）协同工作。

---

🫭 **演示环境：**
👉 [https://atlhyper.com]
ID：`admin` / 密码：`123456`
（部分功能已可使用）

---

### 🚀 核心功能

| 模块           | 功能描述                                                                   |
| -------------- | -------------------------------------------------------------------------- |
| **集群概览**   | 统一展示 Node、Pod、Service、Deployment 等资源信息，含统计卡片与表格视图。 |
| **异常告警**   | 基于事件的诊断、去重与 Slack/邮件通知（含速率限制机制）。                  |
| **资源详情**   | 展示 Pod、Deployment、Namespace 等资源的配置、状态与历史事件。             |
| **操作控制**   | 支持从 Web 界面执行 Pod 重启、Node 隔离（cordon/drain）、资源删除等操作。  |
| **筛选与搜索** | 支持按命名空间、节点、状态、原因等字段过滤，并可进行时间与关键词搜索。     |
| **操作日志**   | 所有操作均以结构化日志记录，并在审计页面可视化展示。                       |
| **配置管理**   | 通过 Web 界面统一管理 Slack、邮件与 Webhook 通知设置。                     |

---

### 🏗️ 系统架构

```plaintext
AtlHyper/
├── atlhyper_master       # 主控中心（外部部署）
├── atlhyper_agent        # 集群内部代理进程
├── atlhyper_metrics      # 节点指标采集器（DaemonSet）
├── atlhyper_aiservice    # AI 智能诊断模块
├── atlhyper_adapter      # 第三方监控数据适配器
├── model/                # 通用数据模型（Pod/Node/Event/Metrics...）
├── utils/                # 工具包（gzip、frame、config）
└── web/                  # 前端管理系统（Vue3 + ElementPlus）
```

---

### 🧩 模块说明

#### 🧠 atlhyper_master（主控中心）

- 汇聚所有 Agent 上报的数据并统一存储。
- 提供 `/ingest/` 接口接收 metrics、events、logs。
- 统一处理 Slack、Mail、Webhook 通知。
- 与 AIService 协同执行智能诊断。
- 执行控制命令（Pod 重启、节点隔离等）。

#### 🛰️ atlhyper_agent（集群代理）

- 采集 Pod、Node、Service、Deployment、Event 等核心资源。
- 与 `atlhyper_metrics` 集成获取节点资源使用率。
- 使用 gzip 压缩并通过 HTTP 向 Master 上报。
- 响应 Master 的操作指令（重启、删除、隔离等）。

#### 📊 atlhyper_metrics（指标采集器）

- 定期采集节点温度、网络速率、磁盘使用率等基础指标。
- 针对轻量环境优化（如 Raspberry Pi）。
- 采集结果经由 Agent 汇聚并上报至 Master。

#### 🤖 atlhyper_aiservice（AI 智能诊断服务）

- 接收 Master 的诊断请求，执行多阶段分析流程：

  - **Stage1：** 初步分析，识别 `needResources`
  - **Stage2：** 从 Master 获取上下文信息
  - **Stage3：** 综合分析输出 RootCause 与 Runbook

- 使用 Gemini 等 LLM 模型（未来将扩展 RAG 能力）。
- 生成结构化诊断报告，辅助 SRE 判断。

#### 🔌 atlhyper_adapter（第三方监控适配器）

- 接收 Prometheus、Zabbix、Datadog、Grafana 等外部监控系统数据。
- 转换为统一结构 `ThirdPartyRecord`。
- 可通过 Agent 或直接上报 Master。
- 示例接口：`/adapter/prometheus/push`, `/adapter/zabbix/alert`

#### 💻 web（前端界面）

- 基于 Vue3 + ElementPlus 实现的单页应用（SPA）。
- 提供集群概览、Pod 详情、事件日志、配置管理等功能页面。
- 通过 Axios 统一请求与响应结构（`code=20000` 表示成功）。
- 使用 CountUp.js 与 ECharts 实现实时统计与图表渲染。

---

### 🧠 AI 流水线结构

| 阶段       | 功能说明                                     |
| ---------- | -------------------------------------------- |
| **Stage1** | 基于事件生成初步诊断，提取 `needResources`。 |
| **Stage2** | 从 Master 获取并组装相关资源上下文。         |
| **Stage3** | 综合分析生成 RootCause、Runbook 与最终结论。 |

---

### 🧭 未来规划

| 阶段       | 目标                                                              |
| ---------- | ----------------------------------------------------------------- |
| **阶段 1** | 通过 `atlhyper_adapter` 集成外部监控数据（Prometheus / Zabbix）。 |
| **阶段 2** | 在 AIService 中引入 RAG / 向量检索，实现知识增强诊断。            |
| **阶段 3** | Master 支持多集群与多租户管理。                                   |
| **阶段 4** | 构建节点与 Pod 自愈引擎（Self-Healing Engine）。                  |

---

### 🧾 总结

> **AtlHyper 是一个模块化、AI 驱动的 Kubernetes 可观测与控制平台。**
> 通过 Master、Agent、Adapter、AI Service 四层结构，实现从异常检测、智能诊断到自动控制的一体化闭环。
