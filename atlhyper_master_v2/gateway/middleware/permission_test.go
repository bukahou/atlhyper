package middleware

import (
	"slices"
	"testing"
)

// 每一级恰好拥有声明的权限：高级继承低级，且不多出任何一项。
func TestPermissionsFor_CumulativeAndExact(t *testing.T) {
	want := map[int][]Permission{}
	var acc []Permission
	for level := RoleMember; level <= RoleAdmin; level++ {
		acc = append(acc, levelGrants[level]...)
		want[level] = slices.Clone(acc)
	}
	for level, perms := range want {
		got := PermissionsFor(level)
		slices.Sort(got)
		slices.Sort(perms)
		if !slices.Equal(got, perms) {
			t.Errorf("level %d: got %v, want %v", level, got, perms)
		}
	}
}

// 0 级（访客 = 成员）只含集群状态与形状：内容级 / 凭据 / 写操作 / 配置一律不得出现。
func TestLevelZero_HasNoSensitivePermission(t *testing.T) {
	sensitive := []Permission{
		PermWorkloadEnvRead, PermLogsRead, PermTracesRead, PermCommandsRead,
		PermDeployHistoryRead, PermGitHubStatusRead, PermAIChat,
		PermSecretRead, PermPodLogsRead, PermClusterOperate, PermAuditRead,
		PermNotifyManage, PermAIAnalyze, PermAISettingsRead,
		PermUserManage, PermAISettingsWrite, PermGitHubManage, PermDeployManage,
	}
	for _, p := range sensitive {
		if Allowed(RoleMember, p) {
			t.Errorf("0 级不得拥有 %s", p)
		}
	}
}

func TestAllowed_Boundaries(t *testing.T) {
	cases := []struct {
		level int
		perm  Permission
		want  bool
	}{
		{RoleMember, PermClusterRead, true},
		{RoleMember, PermLogsStatsRead, true},
		{RoleMember, PermLogsRead, false},
		{RoleViewer, PermLogsRead, true},
		{RoleViewer, PermAIChat, true},
		{RoleViewer, PermSecretRead, false},
		{RoleOperator, PermClusterOperate, true},
		{RoleOperator, PermUserManage, false},
		{RoleAdmin, PermUserManage, true},
		{RoleAdmin, PermClusterRead, true},
		{-1, PermClusterRead, false}, // 越界 level 一律拒绝
		{99, PermClusterRead, false},
	}
	for _, c := range cases {
		if got := Allowed(c.level, c.perm); got != c.want {
			t.Errorf("Allowed(%d, %s) = %v, want %v", c.level, c.perm, got, c.want)
		}
	}
}

// 每个权限只在一级里授予一次（防止表里重复登记导致分档含糊）。
func TestLevelGrants_NoDuplicates(t *testing.T) {
	seen := map[Permission]int{}
	for level, perms := range levelGrants {
		for _, p := range perms {
			if prev, ok := seen[p]; ok {
				t.Errorf("%s 同时出现在 level %d 与 %d", p, prev, level)
			}
			seen[p] = level
		}
	}
}
