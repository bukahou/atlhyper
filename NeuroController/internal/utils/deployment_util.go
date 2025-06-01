// =======================================================================================
// 📄 deployment_util.go
//
// ✨ 功能说明：
//     提供与 Deployment 元信息相关的工具函数，主要用于从 Pod 的 OwnerReference 中
//     解析所属的 Deployment 名称（通过 ReplicaSet 追溯），用于诊断、缩容等场景。
//
// 🛠️ 提供功能：
//     - GetDeploymentNameFromPod(): 从 Pod 对象中获取所属 Deployment 名
//
// 📦 依赖：
//     - k8s.io/api/core/v1
//     - k8s.io/client-go/kubernetes
//
// 📍 使用场景：
//     - Watcher 模块在报警时识别故障 Pod 的归属 Deployment
//     - Scaler 模块执行 scale 操作时确定目标 Deployment 名称
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package utils
