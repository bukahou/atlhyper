// =======================================================================================
// 📄 crash_watcher.go
//
// ✨ 功能说明：
//     本模块使用 Kubernetes 的 Informer 机制监听 Pod 状态，
//     主要用于检测 Pod 是否进入 CrashLoopBackOff、Error 等异常状态。
//     一旦捕捉到异常 Pod，将把其信息传递给日志收集器与告警模块进行后续处理。
//
// 🛠️ 提供功能：
//     - 启动 Pod 级别的共享 Informer
//     - 判断 Pod 是否异常（CrashLoopBackOff、ContainerCannotRun 等）
//     - 调用 log_collector 采集日志并通知 reporter 模块
//
// 📦 依赖：
//     - client-go Informer 机制（k8s.io/client-go/informers）
//     - corev1.Pod 状态判断
//
// 📍 使用场景：
//     - 控制器核心感知模块，自动发现异常 Pod
//     - 搭建 watcher → log_collector → reporter 的监控闭环
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package watcher
