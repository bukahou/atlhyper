package aiops

import "testing"

func TestSplitEntityKey(t *testing.T) {
	cases := []struct {
		key                   string
		namespace, kind, name string
	}{
		{"shop/deployment/api", "shop", "deployment", "api"},
		{"shop/pod/api-7d9f-abcde", "shop", "pod", "api-7d9f-abcde"},
		{"_cluster/node/worker-1", "", "node", "worker-1"},
		{EntityKey("", "node", "worker-2"), "", "node", "worker-2"},
		{"shop/service/a/b", "shop", "service", "a/b"},
		{"unexpected", "", "", "unexpected"},
		{"", "", "", ""},
	}
	for _, c := range cases {
		namespace, kind, name := SplitEntityKey(c.key)
		if namespace != c.namespace || kind != c.kind || name != c.name {
			t.Errorf("SplitEntityKey(%q) = (%q, %q, %q)，期望 (%q, %q, %q)",
				c.key, namespace, kind, name, c.namespace, c.kind, c.name)
		}
	}
}
