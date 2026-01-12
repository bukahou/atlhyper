// source/watcher/register.go
// Watcher 注册表
package watcher

import (
	"log"

	"AtlHyper/atlhyper_agent/source/event/watcher/deployment"
	"AtlHyper/atlhyper_agent/source/event/watcher/endpoint"
	"AtlHyper/atlhyper_agent/source/event/watcher/event"
	"AtlHyper/atlhyper_agent/source/event/watcher/node"
	"AtlHyper/atlhyper_agent/source/event/watcher/pod"
	"AtlHyper/atlhyper_agent/source/event/watcher/service"

	ctrl "sigs.k8s.io/controller-runtime"
)

// ✅ 注册所有 Watcher 到 controller-runtime 的管理器中
//
// 遍历 WatcherRegistry 并调用每个模块的注册方法。
// 如果任意模块注册失败，则终止流程并返回错误。
func RegisterAllWatchers(mgr ctrl.Manager) error {

	for _, w := range WatcherRegistry {
		if err := w.Action(mgr); err != nil {
			log.Printf("❌ 注册 %s 失败: %v", w.Name, err)
			return err
		}
		log.Printf("✅ 注册 %s 成功", w.Name)
	}
	return nil
}

// =======================================================================================
// ✅ Watcher 注册表（集中管理、支持扩展）
//
// 只需将新的 Watcher 模块添加到该列表中，即可实现自动注册。
// =======================================================================================
var WatcherRegistry = []struct {
	Name   string
	Action func(ctrl.Manager) error
}{
	{"PodWatcher", pod.RegisterWatcher},
	{"NodeWatcher", node.RegisterWatcher},
	{"ServiceWatcher", service.RegisterWatcher},
	{"DeploymentWatcher", deployment.RegisterWatcher},
	{"EventWatcher", event.RegisterWatcher},
	{"EndpointWatcher", endpoint.RegisterWatcher},
	// 未来可扩展更多模块，例如：
	// {"PVCWatcher", pvc.RegisterWatcher},
}
