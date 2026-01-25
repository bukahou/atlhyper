// atlhyper_master_v2/tester/interfaces.go
// 测试器接口定义
package tester

import "context"

// Tester 测试器接口
type Tester interface {
	// Name 返回测试器名称
	Name() string

	// Test 执行测试
	// target 是测试目标，如 channelType="slack", clusterID="cluster-1"
	Test(ctx context.Context, target string) Result
}

// Result 测试结果
type Result struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// NewSuccessResult 创建成功结果
func NewSuccessResult(message string) Result {
	return Result{
		Success: true,
		Message: message,
	}
}

// NewSuccessResultWithDetails 创建带详情的成功结果
func NewSuccessResultWithDetails(message string, details map[string]any) Result {
	return Result{
		Success: true,
		Message: message,
		Details: details,
	}
}

// NewFailureResult 创建失败结果
func NewFailureResult(message string) Result {
	return Result{
		Success: false,
		Message: message,
	}
}

// NewFailureResultWithDetails 创建带详情的失败结果
func NewFailureResultWithDetails(message string, details map[string]any) Result {
	return Result{
		Success: false,
		Message: message,
		Details: details,
	}
}
