## 🧠 项目名称：AtlHyper

### 📌 项目定位

AtlHyper 是一个轻量级 Kubernetes 集群可观测性与控制平台，聚焦于节点、Pod、Deployment 等资源的实时监控、异常告警、问题分析与集群操作，结合前后端分离架构，适用于中小规模集群管理者进行本地部署、边缘集群控制或研发环境监控。

本项目采用 **MarstAgent 模式**，即：Agent 常驻部署于 Kubernetes 集群中收集数据并执行操作，而主控程序（Marst）推荐部署在集群外部的 Docker Compose 环境中，通过 HTTP 与各个 Agent 通信，实现集中式控制与多集群支持。

---

🧭 **在线演示地址**
👉 [https://atlhyper.com](https://atlhyper.com) （演示环境，部分功能已部署）
ID：admin
PW：123456

---

### 🚀 项目功能

| 模块         | 功能说明                                                                       |
| ------------ | ------------------------------------------------------------------------------ |
| 集群资源概览 | 提供节点、Pod、Service、Deployment 等核心资源的实时数据卡片与列表视图          |
| 异常告警系统 | 支持基于事件的诊断机制，过滤、去重并发送 Slack/邮件告警（含节流机制）          |
| 资源详情页面 | 支持对 Pod、Deployment、Namespace 等的详细信息展示，包括状态、配置、历史事件等 |
| 控制操作支持 | 支持通过 UI 页面执行如 Pod 重启、节点 cordon/drain、资源删除等操作             |
| 多种筛选器   | 所有表格组件支持字段级筛选（命名空间、状态、节点、原因等）与时间/关键词过滤    |
| 集群日志审计 | 后端记录所有操作行为并展示在操作审计页面                                       |
| 配置管理     | 支持 Web 界面配置邮件、Slack、Webhook 等告警发送方式与行为开关                 |

---

### 🧱 技术架构

#### 🔧 后端（Golang）

- 基于 Gin 框架构建 RESTful 接口
- 使用 controller-runtime/client-go 与 Kubernetes API 通信
- 异常告警引擎模块化，包括告警阈值判断、节流器、轻量格式化等
- 内置 SQLite 数据库（用于日志、告警等记录）
- 支持运行在 Kubernetes 内部或外部 Docker Compose 中

#### 🖼️ 前端（Vue2 + Element UI）

- 重构原始 HTML 页面为 Vue 单页应用（SPA）
- 使用组件化结构（InfoCard、DataTable、EventTable 等）
- 支持分页、下拉筛选、时间范围过滤、关键字搜索
- 使用 CountUp、ECharts 实现卡片统计与图表展示
