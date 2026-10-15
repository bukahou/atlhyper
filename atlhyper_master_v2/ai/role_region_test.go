package ai

import (
	"context"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
)

// region 来自 Master 配置（不来自数据库），只透传给 bedrock
func TestProviderToRoleConfig_RegionOnlyForBedrock(t *testing.T) {
	s := &aiServiceImpl{bedrockRegion: "ap-northeast-1"}

	got := s.providerToRoleConfig(context.Background(), &database.AIProvider{Provider: "bedrock", APIKey: "k", Model: "m"})
	if got.Config.Region != "ap-northeast-1" {
		t.Errorf("bedrock region = %q", got.Config.Region)
	}
	other := s.providerToRoleConfig(context.Background(), &database.AIProvider{Provider: "openai", APIKey: "k", Model: "m"})
	if other.Config.Region != "" {
		t.Errorf("非 bedrock 不应带 region，得到 %q", other.Config.Region)
	}
}
