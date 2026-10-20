// atlhyper_master_v2/ai/github_redact.go
// GitHub 工具读到的文件里，Kubernetes Secret 的值一律替换为 <redacted>
//
// 纵深防御：第一道是仓库范围（github_scope.go）。这里按行处理而不解析 YAML ——
// Helm 模板等「不是合法 YAML」的清单同样要脱敏，解析失败就放行等于失败即开。
package ai

import (
	"regexp"
	"strings"
)

const redactedValue = "<redacted>"

var (
	secretKindLine = regexp.MustCompile(`^\s*kind:\s*["']?Secret["']?\s*(#.*)?$`)
	jsonSecretKind = regexp.MustCompile(`"kind"\s*:\s*"Secret"`)
	docSeparator   = regexp.MustCompile(`^---(\s.*)?$`)
	secretDataKey  = regexp.MustCompile(`^(\s*)(data|stringData):(.*)$`)
	secretGenKey   = regexp.MustCompile(`^(\s*)secretGenerator:\s*(#.*)?$`)
	mapEntry       = regexp.MustCompile(`^(\s*)([^\s#][^:]*?):(\s.*|)$`)
	literalEntry   = regexp.MustCompile(`^(\s*-\s*["']?)([^=\s"']+)=.*$`)
)

// redactSecretManifests 脱敏文件中的 Secret 值：
//   - YAML 中 kind: Secret 的文档：data / stringData 下各键的值（含多行块）
//   - kustomize secretGenerator 的 KEY=VALUE 条目
//   - JSON 格式的 Secret：整个文件不返回
func redactSecretManifests(content string) string {
	if jsonSecretKind.MatchString(content) {
		return "[已隐藏：该文件包含 JSON 格式的 Kubernetes Secret，AI 不读取其内容]"
	}
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	start := 0
	for i, l := range lines {
		if docSeparator.MatchString(l) {
			out = append(out, redactDocument(lines[start:i])...)
			out = append(out, l)
			start = i + 1
		}
	}
	out = append(out, redactDocument(lines[start:])...)
	return strings.Join(out, "\n")
}

// redactDocument 处理一份 YAML 文档（--- 之间）
func redactDocument(doc []string) []string {
	isSecret := false
	for _, l := range doc {
		if secretKindLine.MatchString(l) {
			isSecret = true
			break
		}
	}

	out := make([]string, 0, len(doc))
	for i := 0; i < len(doc); i++ {
		l := doc[i]
		if m := secretDataKey.FindStringSubmatch(l); isSecret && m != nil {
			indent, key, rest := m[1], m[2], strings.TrimSpace(m[3])
			if rest != "" && !strings.HasPrefix(rest, "#") {
				out = append(out, indent+key+": "+redactedValue) // 行内写法 data: {a: b}
				continue
			}
			end := blockEnd(doc, i, len(indent))
			out = append(out, l)
			out = append(out, redactMapBlock(doc[i+1:end])...)
			i = end - 1
			continue
		}
		if m := secretGenKey.FindStringSubmatch(l); m != nil {
			end := blockEnd(doc, i, len(m[1]))
			out = append(out, l)
			for _, b := range doc[i+1 : end] {
				out = append(out, literalEntry.ReplaceAllString(b, "${1}${2}="+redactedValue))
			}
			i = end - 1
			continue
		}
		out = append(out, l)
	}
	return out
}

// blockEnd 返回 doc[keyLine] 这个键的块之后第一行的下标：
// 块 = 空行、缩进更深的行、以及同缩进的序列项（YAML 允许 "key:\n- item"）
func blockEnd(doc []string, keyLine, keyIndent int) int {
	for j := keyLine + 1; j < len(doc); j++ {
		trimmed := strings.TrimSpace(doc[j])
		if trimmed == "" {
			continue
		}
		indent := len(doc[j]) - len(strings.TrimLeft(doc[j], " "))
		if indent > keyIndent || (indent == keyIndent && (trimmed == "-" || strings.HasPrefix(trimmed, "- "))) {
			continue
		}
		return j
	}
	return len(doc)
}

// redactMapBlock 第一层的键保留、值换成 <redacted>；更深的行（多行块的内容）与注释去掉
func redactMapBlock(block []string) []string {
	childIndent := -1
	out := make([]string, 0, len(block))
	for _, l := range block {
		if strings.TrimSpace(l) == "" {
			out = append(out, l)
			continue
		}
		indent := len(l) - len(strings.TrimLeft(l, " "))
		if childIndent < 0 {
			childIndent = indent
		}
		if indent != childIndent {
			continue
		}
		if m := mapEntry.FindStringSubmatch(l); m != nil {
			out = append(out, m[1]+m[2]+": "+redactedValue)
		}
	}
	return out
}
