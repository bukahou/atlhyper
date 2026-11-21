package command

import (
	"context"
	"strings"
	"testing"

	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/command"
)

// 指令带着它所属的集群：与本 Agent 的集群不一致时不执行，并回报一条失败结果。
func TestExecute_ClusterCheck(t *testing.T) {
	cases := []struct {
		name           string
		agentCluster   string
		commandCluster string
		wantExecuted   bool
	}{
		{"集群一致", "prod", "prod", true},
		{"指令没带集群（旧版 Master）", "prod", "", true},
		{"集群不一致", "prod", "staging", false},
		{"大小写不同也算不一致", "prod", "Prod", false},
		{"前缀相同也算不一致", "prod", "prod-2", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			executed := false
			svc := &commandService{
				clusterID: c.agentCluster,
				podRepo:   &mock.PodRepository{},
				genericRepo: &mock.GenericRepository{
					ScaleDeploymentFn: func(ctx context.Context, namespace, name string, replicas int32) error {
						executed = true
						return nil
					},
				},
			}
			cmd := &command.Command{
				ID:        "cmd-1",
				ClusterID: c.commandCluster,
				Action:    command.ActionScale,
				Source:    command.SourceWeb,
				Namespace: "default",
				Name:      "nginx",
				Params:    map[string]any{"replicas": float64(3)},
			}

			result := svc.Execute(context.Background(), cmd)

			if executed != c.wantExecuted {
				t.Fatalf("executed = %v, want %v", executed, c.wantExecuted)
			}
			if result.CommandID != "cmd-1" {
				t.Errorf("结果的指令 ID = %q, want cmd-1（Master 靠它对上原指令）", result.CommandID)
			}
			if result.Success != c.wantExecuted {
				t.Errorf("Success = %v, want %v（%s）", result.Success, c.wantExecuted, result.Error)
			}
			if !c.wantExecuted {
				for _, want := range []string{"集群", c.commandCluster, c.agentCluster} {
					if !strings.Contains(result.Error, want) {
						t.Errorf("错误信息 %q 应包含 %q", result.Error, want)
					}
				}
			}
		})
	}
}

// 集群核对先于其他一切：集群不符的指令连参数都不解析。
func TestExecute_ClusterCheck_PrecedesTargetValidation(t *testing.T) {
	svc := &commandService{clusterID: "prod", podRepo: &mock.PodRepository{}, genericRepo: &mock.GenericRepository{}}
	cmd := &command.Command{ID: "cmd-1", ClusterID: "staging", Action: command.ActionScale, Namespace: "Not A Valid Namespace"}

	result := svc.Execute(context.Background(), cmd)

	if result.Success || !strings.Contains(result.Error, "集群") {
		t.Errorf("应以集群不符为由拒绝，实际 success=%v error=%q", result.Success, result.Error)
	}
}

// 执行方自己按来源允许名单核对：名单外的「来源 × 动作」不执行，回报失败。
func TestExecute_SourceCheck(t *testing.T) {
	cases := []struct {
		name         string
		action       string
		source       string
		wantExecuted bool
	}{
		{"Web 扩缩容", command.ActionScale, command.SourceWeb, true},
		{"AI 扩缩容", command.ActionScale, command.SourceAI, false},
		{"没有来源", command.ActionScale, "", false},
		{"Web 动态查询", command.ActionDynamic, command.SourceWeb, false},
		{"未知动作", "exec", command.SourceWeb, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			executed := false
			generic := &mock.GenericRepository{
				ScaleDeploymentFn: func(ctx context.Context, namespace, name string, replicas int32) error {
					executed = true
					return nil
				},
			}
			svc := &commandService{clusterID: "prod", podRepo: &mock.PodRepository{}, genericRepo: generic}
			cmd := &command.Command{
				ID:        "cmd-1",
				ClusterID: "prod",
				Action:    c.action,
				Namespace: "default",
				Name:      "nginx",
				Source:    c.source,
				Params:    map[string]any{"replicas": float64(3)},
			}

			result := svc.Execute(context.Background(), cmd)

			if executed != c.wantExecuted {
				t.Fatalf("executed = %v, want %v（%s）", executed, c.wantExecuted, result.Error)
			}
			if result.Success != c.wantExecuted {
				t.Errorf("Success = %v, want %v（%s）", result.Success, c.wantExecuted, result.Error)
			}
			if !c.wantExecuted && !strings.Contains(result.Error, "来源") {
				t.Errorf("拒绝原因应说明来源不符，实际 %q", result.Error)
			}
		})
	}
}
