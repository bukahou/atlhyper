// =======================================================================================
// 📄 k8s_client.go
//
// ✨ 功能说明：
//     本文件封装了 Kubernetes 的 client-go 初始化逻辑，统一提供 clientset 实例，
//     供整个控制器项目（如 Watcher、Scaler、Webhook 等模块）访问集群资源。
//     支持在集群内（InCluster）与本地（Out-of-Cluster）两种模式下初始化。
//
// 🛠️ 提供功能：
//     - InitK8sClient(): 初始化 clientset（只需调用一次）
//     - GetClientSet(): 获取已初始化的 kubernetes.Interface 实例
//
// 📦 依赖：
//     - client-go (k8s.io/client-go/kubernetes)
//     - rest config (k8s.io/client-go/rest)
//
// 📍 使用方式：
//     在 controller 启动时调用 InitK8sClient()
//     之后各模块通过 GetClientSet() 获取共享客户端
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package utils
