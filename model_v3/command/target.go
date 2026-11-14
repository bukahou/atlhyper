package command

import (
	"fmt"
	"regexp"
)

// K8s 的资源命名规则（RFC 1123）：
//   - 命名空间：DNS label，小写字母数字与 '-'，首尾为字母数字，最长 63
//   - 资源名：DNS subdomain，由 '.' 分隔的若干 label，最长 253
//
// 指令目标涉及的资源类型（Pod / Deployment / Node / ConfigMap / Secret 等）名字都满足 subdomain 规则。
const (
	maxNamespaceLength = 63
	maxNameLength      = 253
	dnsLabelPattern    = `[a-z0-9]([-a-z0-9]*[a-z0-9])?`
)

var (
	dnsLabelRegexp     = regexp.MustCompile(`^` + dnsLabelPattern + `$`)
	dnsSubdomainRegexp = regexp.MustCompile(`^` + dnsLabelPattern + `(\.` + dnsLabelPattern + `)*$`)
)

// ValidateTarget 校验指令目标的命名空间与名字是否符合 K8s 命名规则。
//
// 空值表示「未指定」，是否必填由各动作自行决定，这里只校验非空值的格式。
// Master 在创建指令时调用，Agent 在执行前再调用一次：目标会成为 K8s API 请求的一部分，
// 两端都只接受合法的资源名。
func ValidateTarget(namespace, name string) error {
	if namespace != "" {
		if len(namespace) > maxNamespaceLength || !dnsLabelRegexp.MatchString(namespace) {
			return fmt.Errorf("invalid target namespace: %q", namespace)
		}
	}
	if name != "" {
		if len(name) > maxNameLength || !dnsSubdomainRegexp.MatchString(name) {
			return fmt.Errorf("invalid target name: %q", name)
		}
	}
	return nil
}
