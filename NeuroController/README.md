# 🕸️ NeuroController · 类控制器插件化系统

NeuroController 是一个神经网络式的 Kubernetes 控制器插件，专为增强系统感知与响应能力设计，旨在打造一个“像蛛网一样敏感”的平台基础模块。它可以监听 Pod 异常、收集诊断信息、触发告警、缩容处理，甚至借助 AI 自动生成建议与修复策略。

---

## 🧠 核心能力

- 🔁 Webhook：接收 DockerHub 镜像更新推送，触发自动策略
- 🧾 结构化日志系统：统一输出 JSON 格式日志，支持 trace.id、时间戳、异常类型等字段，便于接入 Filebeat / Elasticsearch / Loki / Kibana 等可观测平台，实现异常上下文追踪与日志分析。
- 👀 Pod 监听器：自动检测 CrashLoopBackOff 等异常状态
- 🧾 日志诊断器：自动获取 `kubectl describe/logs`
- 📧 告警系统：HTML 格式邮件告警
- 📉 自动缩容：异常 Deployment 缩容至副本数 0
- 🤖 AI 分析模块：基于规则或 LLM 生成错误分类与建议
- 🧠 策略响应器：支持跳过、回滚等扩展策略（预留）

---

## 📁 项目结构

```bash
NeuroController/
├── cmd/                      # 主入口
│   └── controller.go         # 启动所有模块
├── internal/                 # 核心模块
│   ├── webhook/              # 镜像更新 webhook
│   ├── watcher/              # Pod 崩溃监听与诊断
│   ├── reporter/             # 告警模块
│   ├── scaler/               # 自动缩容
│   ├── neuroai/              # AI 建议与规则分析
│   ├── strategy/             # 策略触发与回滚
│   ├── utils/                # 工具集（k8s client、日志等）
│   └── config/               # 配置文件与模板
├── scripts/                  # 辅助脚本
├── Dockerfile                # 构建配置
├── go.mod / go.sum           # Go 依赖管理
└── README.md                 # 项目说明
