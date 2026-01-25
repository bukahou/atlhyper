package service

import (
	modelsvc "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func mapServicePorts(ports []corev1.ServicePort) []modelsvc.ServicePort {
	if len(ports) == 0 {
		return nil
	}
	out := make([]modelsvc.ServicePort, 0, len(ports))
	for _, p := range ports {
		out = append(out, modelsvc.ServicePort{
			Name:        p.Name,
			Protocol:    string(p.Protocol),
			Port:        p.Port,
			TargetPort:  intstrToString(p.TargetPort),
			NodePort:    p.NodePort,
			AppProtocol: stringPtrValue(p.AppProtocol),
		})
	}
	return out
}

func intstrToString(v intstr.IntOrString) string {
	if v.Type == intstr.String {
		return v.StrVal
	}
	return fmtInt32(v.IntVal)
}
