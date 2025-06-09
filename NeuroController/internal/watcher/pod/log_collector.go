// =======================================================================================
// 📄 watcher/pod/log_collector.go
//
// ✨ Description:
//     Collects recent container logs from a Pod for use in diagnostics and alerting.
//     Supports limiting log length by line count, time window, or collecting only
//     the last error segment.
//
// 🛠️ Features:
//     - CollectRecentLogs(pod *corev1.Pod, tail int) → string
//     - Supports specifying container name for multi-container Pods
//
// 📦 Dependencies:
//     - client-go/corev1.PodLogs interface
//     - Standard libraries: context, io, bytes, bufio, etc.
//
// 📍 Usage:
//     - Called by watcher/pod/pod_watcher.go when an abnormal Pod is detected
//     - Used in reporter module to append log content in alert messages
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package pod
