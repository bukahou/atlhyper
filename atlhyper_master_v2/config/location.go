package config

import (
	"fmt"
	"strings"
	"time"
)

// Location 运维时区（MASTER_TIMEZONE）：通知里显示的时间、AI 深度分析的定时窗口（HH:MM）都按它解读。
// 存储与接口一律 UTC，只有这些面向人的地方换算。空 = UTC；名字不合法时返回错误（启动即失败，不静默退回 UTC）。
func (s ServerConfig) Location() (*time.Location, error) {
	name := strings.TrimSpace(s.Timezone)
	if name == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("MASTER_TIMEZONE %q 不是合法的时区名: %w", name, err)
	}
	return loc, nil
}
