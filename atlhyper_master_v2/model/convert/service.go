// atlhyper_master_v2/model/convert/service.go
// model_v2.Service → model.ServiceItem / model.ServiceDetail 转换函数
package convert

import (
	"fmt"
	"strings"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// ServiceItem 转换为列表项（扁平）
func ServiceItem(src *model_v2.Service) model.ServiceItem {
	return model.ServiceItem{
		Name:      src.Summary.Name,
		Namespace: src.Summary.Namespace,
		Type:      src.Summary.Type,
		ClusterIP: src.Summary.ClusterIP,
		Ports:     formatServicePorts(src.Ports),
		Protocol:  firstProtocol(src.Ports),
		Selector:  formatSelector(src.Selector),
		CreatedAt: src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ServiceItems 转换多个 Service 为列表项
func ServiceItems(src []model_v2.Service) []model.ServiceItem {
	if src == nil {
		return []model.ServiceItem{}
	}
	result := make([]model.ServiceItem, len(src))
	for i := range src {
		result[i] = ServiceItem(&src[i])
	}
	return result
}

// ServiceDetail 转换为详情（扁平 + 嵌套）
func ServiceDetail(src *model_v2.Service) model.ServiceDetail {
	ports := make([]model.ServicePortResponse, len(src.Ports))
	for i, p := range src.Ports {
		ports[i] = model.ServicePortResponse{
			Name:        p.Name,
			Protocol:    p.Protocol,
			Port:        p.Port,
			TargetPort:  p.TargetPort,
			NodePort:    p.NodePort,
			AppProtocol: p.AppProtocol,
		}
	}

	return model.ServiceDetail{
		Name:      src.Summary.Name,
		Namespace: src.Summary.Namespace,
		Type:      src.Summary.Type,
		CreatedAt: src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Age:       src.Summary.Age,

		Selector: src.Selector,
		Ports:    ports,

		ClusterIPs:          src.Network.ClusterIPs,
		ExternalIPs:         src.Network.ExternalIPs,
		LoadBalancerIngress: src.Network.LoadBalancerIngress,

		SessionAffinity:       src.Spec.SessionAffinity,
		ExternalTrafficPolicy: src.Network.ExternalTrafficPolicy,
		InternalTrafficPolicy: src.Network.InternalTrafficPolicy,

		IPFamilies:     src.Network.IPFamilies,
		IPFamilyPolicy: src.Network.IPFamilyPolicy,

		Backends: src.Backends,
		Badges:   src.Summary.Badges,
	}
}

// formatServicePorts 格式化端口为字符串，如 "80:30080/TCP→8080, 443/TCP→8443"
func formatServicePorts(ports []model_v2.ServicePort) string {
	if len(ports) == 0 {
		return ""
	}
	parts := make([]string, len(ports))
	for i, p := range ports {
		s := fmt.Sprintf("%d", p.Port)
		if p.NodePort > 0 {
			s = fmt.Sprintf("%d:%d", p.Port, p.NodePort)
		}
		s += "/" + p.Protocol
		if p.TargetPort != "" && p.TargetPort != fmt.Sprintf("%d", p.Port) {
			s += "→" + p.TargetPort
		}
		parts[i] = s
	}
	return strings.Join(parts, ", ")
}

// firstProtocol 获取第一个端口的协议
func firstProtocol(ports []model_v2.ServicePort) string {
	if len(ports) == 0 {
		return ""
	}
	return ports[0].Protocol
}

// formatSelector 格式化 selector 为字符串，如 "app=nginx,tier=frontend"
func formatSelector(sel map[string]string) string {
	if len(sel) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sel))
	for k, v := range sel {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}
