package tools

import (
	"time"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
)

// defaultToolTimeout 经 Agent 执行的工具等结果的上限
const defaultToolTimeout = 30 * time.Second

// Register 把当前部署可用的工具注册进注册表：可选依赖缺失（没配置 GitHub、没有部署历史库）的工具不注册，
// 发给模型的清单里也就没有它（⛔ 不再出现「清单里有、调用才回未启用」）
func Register(reg *toolkit.Registry, d Deps) {
	if d.ToolTimeout == 0 {
		d.ToolTimeout = defaultToolTimeout
	}
	reg.Register(coreTools(d)...)
	if d.DeployHistory != nil {
		reg.Register(deployHistoryTool(d.DeployHistory))
	}
	if d.GitHub != nil {
		reg.Register(githubTools(d.GitHub, d.GitHubScope)...)
	}
}

// coreTools 不依赖可选组件的工具
func coreTools(d Deps) []*toolkit.Tool {
	tools := []*toolkit.Tool{queryClusterTool(d)}
	tools = append(tools, aiopsTools(d)...)
	tools = append(tools, observeTools(d)...)
	return append(tools, eventHistoryTool(d.Events))
}

// catalog 全部工具（含可选的）；只用于列出定义与权限，处理函数不会被调用
func catalog() []*toolkit.Tool {
	tools := coreTools(Deps{})
	tools = append(tools, deployHistoryTool(nil))
	return append(tools, githubTools(nil, nil)...)
}

// PermissionNames 全部工具可能要求的权限（给网关测试核对它们都在权限表里）
func PermissionNames() []string {
	reg := toolkit.NewRegistry()
	reg.Register(catalog()...)
	return reg.Permissions()
}
