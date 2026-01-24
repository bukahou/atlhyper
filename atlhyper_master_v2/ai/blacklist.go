// atlhyper_master_v2/ai/blacklist.go
// AI 黑名单校验
// 在 Tool 执行前校验，禁止写操作、敏感资源、系统命名空间
package ai

import "fmt"

// 禁止的写操作
var forbiddenActions = map[string]bool{
	"scale":        true,
	"restart":      true,
	"delete":       true,
	"delete_pod":   true,
	"exec":         true,
	"cordon":       true,
	"uncordon":     true,
	"drain":        true,
	"update_image": true,
}

// 禁止的命名空间
var forbiddenNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}

// 禁止的资源类型
var forbiddenResources = map[string]bool{
	"Secret": true,
}

// BlacklistCheck 黑名单校验
// 返回 nil 表示通过，返回 error 表示被拒绝
func BlacklistCheck(action, namespace, targetKind string) error {
	// 1. 校验 Action
	if forbiddenActions[action] {
		return fmt.Errorf("操作被禁止: %s 为写操作，AI 不允许执行", action)
	}

	// 2. 校验命名空间
	if forbiddenNamespaces[namespace] {
		return fmt.Errorf("命名空间被禁止: %s 为系统命名空间，AI 不允许访问", namespace)
	}

	// 3. 校验资源类型
	if forbiddenResources[targetKind] {
		return fmt.Errorf("资源类型被禁止: %s 为敏感资源，AI 不允许访问", targetKind)
	}

	return nil
}
