package service

import (
	"sort"
	"strconv"
	"strings"

	mod "AtlHyper/model/k8s"
)

// 是否 Headless：clusterIP 为 "None"
func isHeadless(s mod.Service) bool {
	if strings.EqualFold(s.Summary.ClusterIP, "None") {
		return true
	}
	for _, ip := range s.Network.ClusterIPs {
		if strings.EqualFold(ip, "None") {
			return true
		}
	}
	return false
}

// 是否外部服务：NodePort 或 LoadBalancer
func isExternal(s mod.Service) bool {
	t := pickType(s)
	return t == "NodePort" || t == "LoadBalancer"
}

// 类型优先取 summary.type（若空则 spec.type）
func pickType(s mod.Service) string {
	if s.Summary.Type != "" {
		return s.Summary.Type
	}
	return s.Spec.Type
}

// 表格中的 ClusterIP 单值展示
func firstClusterIPForTable(s mod.Service) string {
	if isHeadless(s) {
		return "None"
	}
	if s.Summary.ClusterIP != "" {
		return s.Summary.ClusterIP
	}
	if len(s.Network.ClusterIPs) > 0 {
		return s.Network.ClusterIPs[0]
	}
	return "-"
}

// 表格中的 ports 文本（含 NodePort/LB 场景）
func formatPortsForTable(s mod.Service) string {
	if len(s.Ports) == 0 {
		return "-"
	}
	out := make([]string, 0, len(s.Ports))
	t := pickType(s)
	for _, p := range s.Ports {
		item := ""
		// <port>:<targetPort>
		if p.TargetPort != "" {
			item = strings.TrimSpace(strings.Join([]string{itoa32(p.Port), p.TargetPort}, ":"))
		} else {
			item = itoa32(p.Port)
		}
		// NodePort/LB 展示 nodePort
		if (t == "NodePort" || t == "LoadBalancer") && p.NodePort > 0 {
			item += "(" + itoa32(p.NodePort) + ")"
		}
		out = append(out, item)
	}
	return strings.Join(out, ", ")
}

// 协议去重拼接
func joinProtocols(s mod.Service) string {
	set := map[string]struct{}{}
	for _, p := range s.Ports {
		if p.Protocol == "" {
			continue
		}
		set[p.Protocol] = struct{}{}
	}
	if len(set) == 0 {
		return "-"
	}
	arr := make([]string, 0, len(set))
	for k := range set {
		arr = append(arr, k)
	}
	sort.Strings(arr)
	return strings.Join(arr, ", ")
}

// Selector 格式化 "k=v" 列表
func formatSelectorKV(sel map[string]string) string {
	if len(sel) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(sel))
	for k := range sel {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+sel[k])
	}
	return strings.Join(parts, ", ")
}

func itoa32(v int32) string {
	return strconv.FormatInt(int64(v), 10)
}


func inferBadges(s mod.Service) []string {
    badges := []string{}

    // Headless
    if strings.EqualFold(s.Summary.ClusterIP, "None") {
        badges = append(badges, "Headless")
    }

    // LoadBalancer
    if strings.EqualFold(s.Summary.Type, "LoadBalancer") {
        badges = append(badges, "LoadBalancer")
    }

    // NodePort
    if strings.EqualFold(s.Summary.Type, "NodePort") {
        badges = append(badges, "NodePort")
    }

    // ExternalName
    if strings.EqualFold(s.Summary.Type, "ExternalName") {
        badges = append(badges, "ExternalName")
    }

    // NoSelector
    if !s.Summary.HasSelector && len(s.Selector) == 0 {
        badges = append(badges, "NoSelector")
    }

    return badges
}
