package command

import (
	"errors"
	"fmt"
)

// 指令来源：由 Master 内部各条下发路径自己填写，⛔ 不取自请求参数
const (
	SourceWeb      = "web"      // Web 页面的操作与查询：每个动作有自己的接口，权限按接口（即按动作）声明
	SourceAI       = "ai"       // AI 工具调用：只许读
	SourceDeployer = "deployer" // GitOps 部署器：只许应用清单
)

// ErrActionNotPermitted 动作不在允许名单里，或不允许由该来源发起
var ErrActionNotPermitted = errors.New("指令动作不允许由该来源发起")

// actionSources 每个动作允许由哪些来源发起。
//
// Master 下发前、Agent 执行前都按这同一份核对：名单外的动作、名单外的来源一律拒绝。
// 新增动作时在这里声明来源，⛔ 不在别处另写一份。
var actionSources = map[string][]string{
	ActionScale:       {SourceWeb},
	ActionRestart:     {SourceWeb},
	ActionDelete:      {SourceWeb},
	ActionUpdateImage: {SourceWeb},
	ActionCordon:      {SourceWeb},
	ActionUncordon:    {SourceWeb},

	ActionGetLogs:      {SourceWeb, SourceAI},
	ActionGetConfigMap: {SourceWeb, SourceAI},
	ActionGetSecret:    {SourceWeb},
	ActionDynamic:      {SourceAI}, // 动作本身只读：Agent 另按动词与资源类型的允许名单核对

	ActionApplyManifests: {SourceDeployer},

	ActionQueryTraces:      {SourceWeb, SourceAI},
	ActionQueryTraceDetail: {SourceWeb},
	ActionQueryLogs:        {SourceWeb, SourceAI},
	ActionQueryMetrics:     {SourceWeb},
}

// IsValidAction 是否是名单里的动作
func IsValidAction(action string) bool {
	_, ok := actionSources[action]
	return ok
}

// CheckSource 核对动作是否允许由该来源发起
func CheckSource(action, source string) error {
	sources, ok := actionSources[action]
	if !ok {
		return fmt.Errorf("%w：未知动作 %q", ErrActionNotPermitted, action)
	}
	for _, allowed := range sources {
		if source == allowed {
			return nil
		}
	}
	return fmt.Errorf("%w：动作 %q 不允许由来源 %q 发起", ErrActionNotPermitted, action, source)
}
