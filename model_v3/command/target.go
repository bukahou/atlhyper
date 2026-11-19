package command

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation"
)

// K8s 的资源命名规则（RFC 1123），用 apimachinery 的同一套校验：
//   - 命名空间：DNS label
//   - 资源名：DNS subdomain（指令目标涉及的 Pod / Deployment / Node / ConfigMap / Secret 都满足）

// ValidateTarget 校验指令目标的命名空间与名字是否符合 K8s 命名规则。
//
// 空值表示「未指定」，是否必填由各动作自行决定，这里只校验非空值的格式。
// Master 在创建指令时调用，Agent 在执行前再调用一次：目标会成为 K8s API 请求的一部分，
// 两端都只接受合法的资源名。
func ValidateTarget(namespace, name string) error {
	if namespace != "" {
		if len(validation.IsDNS1123Label(namespace)) > 0 {
			return fmt.Errorf("invalid target namespace: %q", namespace)
		}
	}
	if name != "" {
		if len(validation.IsDNS1123Subdomain(name)) > 0 {
			return fmt.Errorf("invalid target name: %q", name)
		}
	}
	return nil
}
