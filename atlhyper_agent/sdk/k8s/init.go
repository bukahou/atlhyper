// sdk/k8s/init.go
// K8s Provider 自动注册
package k8s

import "AtlHyper/atlhyper_agent/sdk"

func init() {
	sdk.RegisterProvider("kubernetes", NewK8sProvider)
}
