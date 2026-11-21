package scheduler

import (
	"testing"

	"AtlHyper/model_v3/command"
)

func TestCommandTarget(t *testing.T) {
	cases := []struct {
		cmd  command.Command
		want string
	}{
		{command.Command{Kind: "Deployment", Namespace: "shop", Name: "web"}, "Deployment shop/web"},
		{command.Command{Kind: "Node", Name: "node-a"}, "Node node-a"},
		{command.Command{Action: command.ActionQueryTraces}, ""}, // 查询类指令没有目标
	}
	for _, c := range cases {
		if got := commandTarget(&c.cmd); got != c.want {
			t.Errorf("commandTarget(%+v) = %q, want %q", c.cmd, got, c.want)
		}
	}
}
