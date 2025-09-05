## 🌐 Language / 言語 / 语言

- 🇺🇸 English (current)
- 🇯🇵 [日本語 / Japanese](./README.ja.md)
- 🇨🇳 [简体中文 / Simplified Chinese](./README.zh-CN.md)

## 🧠 Project Name: AtlHyper

### 📌 Project Overview

**AtlHyper** is a lightweight Kubernetes observability and control platform designed for real-time monitoring, anomaly detection, analysis, and direct interaction with key resources such as Nodes, Pods, and Deployments. Built with a frontend-backend decoupled architecture, it targets small-to-medium scale clusters for local deployment, edge cluster control, and development environment supervision.

The system adopts a **MarstAgent architecture**:

- **Agent**: Deployed inside each Kubernetes cluster, responsible for data collection and local operations.
- **Master (Marst)**: Recommended to run externally via Docker Compose, communicating with Agents via HTTP, enabling centralized control and multi-cluster support.

---

🫭 Online Demo:
👉 [https://atlhyper.com](https://atlhyper.com)
(_Demo environment with partial functionality enabled_)
**ID**: admin
**Password**: 123456

---

### 🚀 Key Features

| Module              | Description                                                                      |
| ------------------- | -------------------------------------------------------------------------------- |
| Cluster Overview    | Real-time cards and lists for Nodes, Pods, Services, and Deployments             |
| Alert System        | Event-based anomaly diagnosis, deduplication, Slack/email alerts with throttling |
| Resource Detail     | Drill-down views for Pods, Deployments, Namespaces, with status and history      |
| Operational Control | UI-based controls: Pod restart, node cordon/drain, delete resources              |
| Filtering Support   | Field-level filters (namespace, status, node, reason) and time/keyword search    |
| Action Audit        | Logs all backend operations, visible in the audit trail page                     |
| Configuration Panel | Web-based configuration of alert channels (Slack, email, webhook) and toggles    |

---

### 🛠️ Architecture Overview

#### 🔧 Backend (Golang)

- Built with Gin for RESTful APIs
- Uses controller-runtime/client-go to interact with the Kubernetes API
- Modular alert engine: threshold checks, throttlers, lightweight formatters
- Embedded SQLite database for logs, alerts, configuration persistence
- Runs either inside Kubernetes or externally via Docker Compose

#### 📺 Frontend (Vue2 + Element UI)

- Migrated from legacy HTML to Vue SPA
- Component-based layout (InfoCard, DataTable, EventTable, etc.)
- Supports pagination, dropdown filters, time range, and keyword search
- Uses CountUp and ECharts for dynamic cards and charts

---
