package ai

import (
	"strings"
	"testing"
)

func TestRedactSecretManifests_LeavesOtherFilesAlone(t *testing.T) {
	for _, in := range []string{
		"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\ndata:\n  LOG_LEVEL: info\n",
		"package main\n\nfunc main() {}\n",
		"# README\n\nkind of secret sauce: not a manifest\n",
	} {
		if got := redactSecretManifests(in); got != in {
			t.Errorf("不含 Secret 的文件不应改动：\n%s\n=>\n%s", in, got)
		}
	}
}

func TestRedactSecretManifests_SecretValues(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{
			name: "data 与 stringData 只换值、保留键名",
			in: "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app-secret\ntype: Opaque\n" +
				"data:\n  DB_PASSWORD: c2VjcmV0\n  API_KEY: a2V5\nstringData:\n  TOKEN: plain-token\n",
			want: "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app-secret\ntype: Opaque\n" +
				"data:\n  DB_PASSWORD: <redacted>\n  API_KEY: <redacted>\nstringData:\n  TOKEN: <redacted>\n",
		},
		{
			name: "多行块整段换掉，块结束后的字段原样",
			in:   "kind: Secret\nstringData:\n  config.yaml: |\n    user: admin\n    password: hunter2\n  other: x\nmetadata:\n  name: s\n",
			want: "kind: Secret\nstringData:\n  config.yaml: <redacted>\n  other: <redacted>\nmetadata:\n  name: s\n",
		},
		{
			name: "多文档只处理 Secret 那一份",
			in:   "kind: ConfigMap\ndata:\n  LEVEL: info\n---\nkind: Secret\ndata:\n  PASSWORD: aHVudGVyMg==\n",
			want: "kind: ConfigMap\ndata:\n  LEVEL: info\n---\nkind: Secret\ndata:\n  PASSWORD: <redacted>\n",
		},
		{
			name: "行内写法",
			in:   "kind: Secret\ndata: {a: YQ==, b: Yg==}\n",
			want: "kind: Secret\ndata: <redacted>\n",
		},
		{
			name: "kind 带引号",
			in:   "kind: \"Secret\"\ndata:\n  a: YQ==\n",
			want: "kind: \"Secret\"\ndata:\n  a: <redacted>\n",
		},
		{
			name: "Helm 模板",
			in:   "kind: Secret\nmetadata:\n  name: {{ .Release.Name }}\ndata:\n  password: {{ .Values.password | b64enc | quote }}\n",
			want: "kind: Secret\nmetadata:\n  name: {{ .Release.Name }}\ndata:\n  password: <redacted>\n",
		},
		{
			name: "kustomize secretGenerator 的 literals",
			in: "secretGenerator:\n- name: app\n  literals:\n  - DB_PASSWORD=hunter2\n  - \"USER=admin\"\n" +
				"configMapGenerator:\n- name: cfg\n  literals:\n  - LEVEL=info\n",
			want: "secretGenerator:\n- name: app\n  literals:\n  - DB_PASSWORD=<redacted>\n  - \"USER=<redacted>\n" +
				"configMapGenerator:\n- name: cfg\n  literals:\n  - LEVEL=info\n",
		},
	}
	for _, c := range cases {
		if got := redactSecretManifests(c.in); got != c.want {
			t.Errorf("%s:\n got:\n%s\nwant:\n%s", c.name, got, c.want)
		}
	}
}

func TestRedactSecretManifests_JSONSecretWithheld(t *testing.T) {
	got := redactSecretManifests(`{"apiVersion":"v1","kind": "Secret","data":{"a":"YQ=="}}`)
	if strings.Contains(got, "YQ==") || !strings.Contains(got, "已隐藏") {
		t.Errorf("JSON 格式的 Secret 应整体隐藏，got %q", got)
	}
}
