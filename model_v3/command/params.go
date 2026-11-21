package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/distribution/reference"
	"k8s.io/apimachinery/pkg/util/validation"
)

// 指令参数的共享定义。
//
// Command.Params 在传输上是一个 map，Master 填、Agent 读。两端各写各的键名很容易对不上
// （一端 tail_lines、另一端 tailLines，参数就静默失效）。带参数的指令在这里定义结构体：
// Master 用 ToParams 生成，Agent 用 ParseParams 解析，键名只有这一处。

// GetLogsParams get_logs 指令的参数
type GetLogsParams struct {
	Container    string `json:"container,omitempty"`    // 容器名，空则由 Agent 选主容器
	TailLines    int64  `json:"tailLines,omitempty"`    // 返回最后 N 行
	SinceSeconds int64  `json:"sinceSeconds,omitempty"` // 只返回最近 N 秒
	Timestamps   bool   `json:"timestamps,omitempty"`   // 每行带时间戳
	Previous     bool   `json:"previous,omitempty"`     // 读上一个容器实例的日志
}

// ToParams 转成 Command.Params
func (p GetLogsParams) ToParams() map[string]any {
	return toParams(p)
}

// ParseParams 把 Command.Params 解析进参数结构体
func ParseParams(params map[string]any, into any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}

func toParams(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	params := map[string]any{}
	if err := json.Unmarshal(raw, &params); err != nil {
		return map[string]any{}
	}
	return params
}

// MaxScaleReplicas 扩缩容允许的最大副本数：防止手误或越权把一个服务扩到集群装不下
const MaxScaleReplicas int32 = 100

// ScaleParams scale 指令的参数。
// 副本数是指针：缺了就是缺了，⛔ 不能当成 0 去执行（那等于把服务缩到零）
type ScaleParams struct {
	Replicas *int32 `json:"replicas"`
}

// ToParams 转成 Command.Params
func (p ScaleParams) ToParams() map[string]any {
	return toParams(p)
}

// Validate 发送前与执行前各调一次
func (p ScaleParams) Validate() error {
	if p.Replicas == nil {
		return fmt.Errorf("replicas is required")
	}
	if *p.Replicas < 0 || *p.Replicas > MaxScaleReplicas {
		return fmt.Errorf("replicas must be between 0 and %d, got %d", MaxScaleReplicas, *p.Replicas)
	}
	return nil
}

// UpdateImageParams update_image 指令的参数
type UpdateImageParams struct {
	Container string `json:"container"` // 必填：改哪个容器由调用方明确指定，不由 Agent 猜
	Image     string `json:"image"`
}

// ToParams 转成 Command.Params
func (p UpdateImageParams) ToParams() map[string]any {
	return toParams(p)
}

// Validate 容器名按 K8s 规则、镜像按 OCI 引用规则解析，不合格的不发也不执行
func (p UpdateImageParams) Validate() error {
	if p.Container == "" {
		return fmt.Errorf("container is required")
	}
	if errs := validation.IsDNS1123Label(p.Container); len(errs) > 0 {
		return fmt.Errorf("invalid container name %q: %s", p.Container, strings.Join(errs, "; "))
	}
	if p.Image == "" {
		return fmt.Errorf("image is required")
	}
	if _, err := reference.ParseNormalizedNamed(p.Image); err != nil {
		return fmt.Errorf("invalid image reference %q: %w", p.Image, err)
	}
	return nil
}
