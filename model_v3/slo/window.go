package slo

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Windows Agent 预先算好的 SLO 统计窗口（只有这些，按长度排序）。
// Agent 的采集配置、Master 的接口参数校验、AI 工具的参数枚举都以这一份为准；
// 前端 lib/time-range.ts 的 SLO_WINDOWS 需与它一致。
var Windows = []string{"1h", "6h", "24h", "3d", "7d"}

// DefaultWindow 没指定窗口时用的窗口
const DefaultWindow = "24h"

// ErrUnknownWindow 不是 Agent 预先算好的窗口
var ErrUnknownWindow = errors.New("不是可用的 SLO 窗口")

// CheckWindow 窗口是否在列表里；不在时返回的错误里列出可选值
func CheckWindow(window string) error {
	if slices.Contains(Windows, window) {
		return nil
	}
	return fmt.Errorf("%w：%q，可选 %s", ErrUnknownWindow, window, strings.Join(Windows, " / "))
}
