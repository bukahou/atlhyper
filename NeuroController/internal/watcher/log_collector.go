// =======================================================================================
// 📄 log_collector.go
//
// ✨ 功能说明：
//     本模块负责自动获取异常 Pod 的诊断信息，包括：
//     - `kubectl describe pod` 的详细状态
//     - `kubectl logs` 的最近日志片段（默认采集最后 N 行）
//     并将这些信息打包为可读格式，传递给告警模块进行邮件渲染与发送。
//
// 🛠️ 提供功能：
//     - FetchPodDescribe(): 获取 Pod 的详细描述信息
//     - FetchPodLogs(): 获取 Pod 最近日志（支持 tailN）
//     - CollectDebugInfo(): 聚合 describe + logs 为告警内容
//
// 📦 依赖：
//     - k8s.io/client-go/kubernetes
//     - corev1.Pod / v1.PodLogOptions
//
// 📍 使用场景：
//     - 被 crash_watcher 调用，作为诊断数据源
//     - 后续 reporter 邮件内容的核心构成部分
//
// 🛠️ 可扩展点：
//     - 日志行数配置（tailLines）
//     - 多容器 Pod 支持
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package watcher
