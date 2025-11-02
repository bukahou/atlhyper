## 🧠 Project Name: AtlHyper

### 📌 Project Overview

**AtlHyper** is a lightweight observability and control platform designed for Kubernetes clusters. It provides real-time monitoring of resources such as Nodes, Pods, and Deployments, performs anomaly detection, alerting, AI-driven diagnosis, and enables operational control actions — all within a unified system. With its frontend-backend separation architecture, AtlHyper is ideal for small-to-medium-sized clusters, edge environments, and development/testing deployments.

The project adopts a **Master-Agent model**. The Agent runs inside each Kubernetes cluster, collecting and transmitting data, while the Master — typically deployed externally (e.g., via Docker Compose) — handles centralized management, control, and coordination. The Master communicates with each Agent via HTTP and integrates seamlessly with the AI diagnosis module (AI Service) and the new external data ingestion module (Adapter).

---

🫭 **Demo Environment:**
👉 [https://atlhyper.com](https://atlhyper.com)
ID: `admin` / PW: `123456`
(Some features are live in the demo environment.)

---

### 🚀 Key Features

| Module                       | Description                                                                                          |
| ---------------------------- | ---------------------------------------------------------------------------------------------------- |
| **Cluster Overview**         | Unified dashboard displaying Nodes, Pods, Services, and Deployments with summary cards and tables.   |
| **Anomaly Alerts**           | Event-driven diagnosis, deduplication, and alerting via Slack and email (with rate limiting).        |
| **Resource Details**         | Detailed configuration and historical events for Pods, Deployments, Namespaces, and other resources. |
| **Operational Control**      | Execute Pod restarts, Node cordon/drain, and resource deletion from the web UI.                      |
| **Filtering & Search**       | Table-level filters by Namespace, Node, Reason, Status, plus time and keyword search.                |
| **Operation Logging**        | All actions are recorded as structured logs and displayed in an audit view.                          |
| **Configuration Management** | Manage Slack, Mail, and Webhook notification settings from the web UI.                               |

---

### 🏗️ System Architecture

```plaintext
AtlHyper/
├── atlhyper_master       # Central control process (external deployment)
├── atlhyper_agent        # Cluster-side agent process
├── atlhyper_metrics      # Node-level metrics collector (DaemonSet)
├── atlhyper_aiservice    # AI diagnosis and analysis module
├── atlhyper_adapter      # Third-party monitoring data adapter
├── model/                # Shared data models (Pod/Node/Event/Metrics...)
├── utils/                # Common utilities (gzip, frame, config)
└── web/                  # Vue3 + ElementPlus based management frontend
```

---

### 🧩 Module Descriptions

#### 🧠 atlhyper_master (Control Center)

- Aggregates and stores data from all Agents.
- Provides `/ingest/` endpoints for metrics, events, and logs.
- Integrates Slack/Mail/Webhook notifications.
- Coordinates with AIService for intelligent diagnosis.
- Executes control operations (Pod restart, Node isolation, etc.).

#### 🛰️ atlhyper_agent (Cluster Agent)

- Collects Pod, Node, Service, Deployment, and Event data.
- Integrates with `atlhyper_metrics` for Node usage and status.
- Compresses and transmits data to Master via gzip HTTP.
- Executes commands from the Master for operational control.

#### 📊 atlhyper_metrics (Metrics Collector)

- Collects temperature, network speed, and disk usage metrics per node.
- Designed for lightweight environments like Raspberry Pi clusters.
- Reports metrics snapshots to the Agent for aggregation.

#### 🤖 atlhyper_aiservice (AI Diagnosis Service)

- Receives diagnostic requests from the Master.
- Implements a multi-stage analysis pipeline:

  - **Stage1:** Preliminary analysis, identifying `needResources`.
  - **Stage2:** Fetching contextual resource data from Master.
  - **Stage3:** Final reasoning, producing RootCause and Runbook.

- Uses Gemini or other LLMs (RAG support planned).
- Outputs structured diagnostic reports.

#### 🔌 atlhyper_adapter (Third-Party Monitoring Adapter)

- Receives data from Prometheus, Zabbix, Datadog, Grafana, etc.
- Normalizes inputs into a standard `ThirdPartyRecord` format.
- Transmits data to Master directly or via Agent.
- Example endpoints: `/adapter/prometheus/push`, `/adapter/zabbix/alert`.

#### 💻 web (Frontend)

- Built with Vue3 + ElementPlus (SPA structure).
- Displays cluster overview, Pod details, event logs, and configuration UI.
- Unified API management via Axios (success code `20000`).
- Utilizes CountUp.js and ECharts for real-time statistics and visualizations.

---

### 🧠 AI Pipeline Structure

| Stage      | Description                                                                            |
| ---------- | -------------------------------------------------------------------------------------- |
| **Stage1** | Generates preliminary AI analysis based on recent events and extracts `needResources`. |
| **Stage2** | Fetches and assembles relevant resource context from Master.                           |
| **Stage3** | Produces RootCause, Runbook, and final summarized diagnosis.                           |

---

### 🧭 Future Roadmap

| Phase       | Objective                                                                            |
| ----------- | ------------------------------------------------------------------------------------ |
| **Phase 1** | Integrate third-party data through `atlhyper_adapter` (Prometheus / Zabbix).         |
| **Phase 2** | Introduce RAG / Embedding search into AIService for knowledge-augmented diagnostics. |
| **Phase 3** | Add multi-cluster and multi-tenant management to Master.                             |
| **Phase 4** | Implement Node/Pod self-healing engine (Autonomous Control Plane).                   |

---

### 🧾 Summary

> **AtlHyper is a modular, AI-enhanced Kubernetes observability and control platform.**
> Through its four-layer architecture — **Master, Agent, Adapter, and AI Service** — it unifies anomaly detection, intelligent diagnosis, and operational automation across distributed clusters.
