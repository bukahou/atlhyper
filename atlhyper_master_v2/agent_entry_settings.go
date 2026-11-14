// atlhyper_master_v2/agent_entry_settings.go
// Agent 入口（AgentSDK）的启动配置校验
package atlhyper_master_v2

import (
	"fmt"

	"AtlHyper/atlhyper_master_v2/agentsdk"
	"AtlHyper/atlhyper_master_v2/config"
)

// agentEntrySettings 校验过的 Agent 入口配置
type agentEntrySettings struct {
	authMode         agentsdk.AuthMode
	maxSnapshotBytes int64
}

// resolveAgentEntrySettings 把配置里的字符串与数字换成 AgentSDK 用的取值。
//
// 认证模式写错时启动失败，而不是悄悄落到某一档：
// 运维以为开了强制、实际只在告警（或反过来把所有 Agent 拒之门外），都比启动不了更难发现。
func resolveAgentEntrySettings(cfg config.AgentSDKConfig) (agentEntrySettings, error) {
	mode, err := agentsdk.ParseAuthMode(cfg.AuthMode)
	if err != nil {
		return agentEntrySettings{}, fmt.Errorf("MASTER_AGENT_AUTH_MODE: %w", err)
	}
	if cfg.MaxSnapshotMB <= 0 {
		return agentEntrySettings{}, fmt.Errorf("MASTER_AGENTSDK_MAX_SNAPSHOT_MB 必须是正整数，实际 %d", cfg.MaxSnapshotMB)
	}
	return agentEntrySettings{
		authMode:         mode,
		maxSnapshotBytes: int64(cfg.MaxSnapshotMB) << 20,
	}, nil
}
