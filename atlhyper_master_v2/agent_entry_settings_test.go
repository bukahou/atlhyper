package atlhyper_master_v2

import (
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/agentsdk"
	"AtlHyper/atlhyper_master_v2/config"
)

func TestResolveAgentEntrySettings(t *testing.T) {
	cases := []struct {
		name      string
		cfg       config.AgentSDKConfig
		wantMode  agentsdk.AuthMode
		wantBytes int64
		wantErr   string // 非空 = 期望报错，且错误信息里要点名这个环境变量
	}{
		{"默认值", config.AgentSDKConfig{AuthMode: "warn", MaxSnapshotMB: 256}, agentsdk.AuthModeWarn, 256 << 20, ""},
		{"关闭", config.AgentSDKConfig{AuthMode: "off", MaxSnapshotMB: 1}, agentsdk.AuthModeOff, 1 << 20, ""},
		{"强制", config.AgentSDKConfig{AuthMode: "enforce", MaxSnapshotMB: 64}, agentsdk.AuthModeEnforce, 64 << 20, ""},
		{"未知模式", config.AgentSDKConfig{AuthMode: "strict", MaxSnapshotMB: 256}, "", 0, "MASTER_AGENT_AUTH_MODE"},
		{"大小写不符也算未知", config.AgentSDKConfig{AuthMode: "Enforce", MaxSnapshotMB: 256}, "", 0, "MASTER_AGENT_AUTH_MODE"},
		{"空模式", config.AgentSDKConfig{AuthMode: "", MaxSnapshotMB: 256}, "", 0, "MASTER_AGENT_AUTH_MODE"},
		{"上限为零", config.AgentSDKConfig{AuthMode: "warn", MaxSnapshotMB: 0}, "", 0, "MASTER_AGENTSDK_MAX_SNAPSHOT_MB"},
		{"上限为负", config.AgentSDKConfig{AuthMode: "warn", MaxSnapshotMB: -5}, "", 0, "MASTER_AGENTSDK_MAX_SNAPSHOT_MB"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolveAgentEntrySettings(c.cfg)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("期望报错并点名 %s，实际 %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("不应报错: %v", err)
			}
			if got.authMode != c.wantMode || got.maxSnapshotBytes != c.wantBytes {
				t.Errorf("got %+v, want mode=%s bytes=%d", got, c.wantMode, c.wantBytes)
			}
		})
	}
}
