package cluster

import "strings"

// 部署工具会把资源的「整份清单副本」写在注解里（kubectl apply 的 last-applied-configuration 最常见）。
//
// 这类注解把完整清单再存一遍：体积大，内容与快照里的结构化字段完全重复，
// 展示与分析都只使用结构化字段。因此不随快照上报，Master 收到快照时也不保留。
//
// 判定分三层，命中任一层即视为清单副本：
//  1. 已知键名：值经过压缩或编码、从内容上认不出来的那几种
//  2. 键名后缀：各家沿用的 last-applied 命名（kubectl / k0s / 其他部署器）
//  3. 值的内容：是一份序列化的 K8s 对象 —— 不依赖键名，新工具换个名字也拦得住
var manifestCopyAnnotations = map[string]struct{}{
	"kapp.k14s.io/original":           {}, // Carvel kapp
	"objectset.rio.cattle.io/applied": {}, // Rancher wrangler（gzip + base64）
}

var manifestCopySuffixes = []string{
	"/last-applied-configuration", // kubectl.kubernetes.io/ · k0s.k0sproject.io/ 等
	"/last-applied",               // banzaicloud.com/ 等
}

// isManifestCopy 判断一条注解是否是整份清单副本
func isManifestCopy(key, value string) bool {
	if _, ok := manifestCopyAnnotations[key]; ok {
		return true
	}
	for _, suffix := range manifestCopySuffixes {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return looksLikeSerializedObject(value)
}

// looksLikeSerializedObject 值是否像一份 JSON 序列化的 K8s 对象（同时带 apiVersion 与 kind 字段）
func looksLikeSerializedObject(value string) bool {
	if !strings.HasPrefix(strings.TrimLeft(value, " \t\r\n"), "{") {
		return false
	}
	return strings.Contains(value, `"apiVersion"`) && strings.Contains(value, `"kind"`)
}

// ReportableAnnotations 返回可以随快照上报的注解：去掉整份清单副本，其余原样保留。
//
// 不修改传入的 map；没有需要去掉的键时直接返回原 map，全部被去掉时返回 nil。
func ReportableAnnotations(src map[string]string) map[string]string {
	dropped := 0
	for key, value := range src {
		if isManifestCopy(key, value) {
			dropped++
		}
	}
	if dropped == 0 {
		return src
	}
	if dropped == len(src) {
		return nil
	}
	kept := make(map[string]string, len(src)-dropped)
	for key, value := range src {
		if !isManifestCopy(key, value) {
			kept[key] = value
		}
	}
	return kept
}

// DropManifestCopyAnnotations 就地删除快照内所有资源上的整份清单副本注解。
//
// Master 在接收快照时调用：不依赖上报方已经做过清理。
// 覆盖范围由 annotations_test.go 里的反射测试守护，新增带 Annotations 的类型时必须同步这里。
func (s *ClusterSnapshot) DropManifestCopyAnnotations() {
	if s == nil {
		return
	}
	for i := range s.Pods {
		dropManifestCopies(s.Pods[i].Annotations)
	}
	for i := range s.Deployments {
		dropManifestCopies(s.Deployments[i].Annotations)
		dropManifestCopies(s.Deployments[i].Template.Annotations)
	}
	for i := range s.StatefulSets {
		dropManifestCopies(s.StatefulSets[i].Annotations)
		dropManifestCopies(s.StatefulSets[i].Template.Annotations)
	}
	for i := range s.DaemonSets {
		dropManifestCopies(s.DaemonSets[i].Annotations)
		dropManifestCopies(s.DaemonSets[i].Template.Annotations)
	}
	for i := range s.Jobs {
		dropManifestCopies(s.Jobs[i].Template.Annotations)
	}
	for i := range s.CronJobs {
		dropManifestCopies(s.CronJobs[i].Template.Annotations)
	}
	for i := range s.Services {
		dropManifestCopies(s.Services[i].Annotations)
	}
	for i := range s.Ingresses {
		dropManifestCopies(s.Ingresses[i].Annotations)
	}
	for i := range s.Namespaces {
		namespace := &s.Namespaces[i]
		dropManifestCopies(namespace.Annotations)
		for j := range namespace.Quotas {
			dropManifestCopies(namespace.Quotas[j].Annotations)
		}
		for j := range namespace.LimitRanges {
			dropManifestCopies(namespace.LimitRanges[j].Annotations)
		}
	}
	for i := range s.ResourceQuotas {
		dropManifestCopies(s.ResourceQuotas[i].Annotations)
	}
	for i := range s.LimitRanges {
		dropManifestCopies(s.LimitRanges[i].Annotations)
	}
	for i := range s.NetworkPolicies {
		dropManifestCopies(s.NetworkPolicies[i].Annotations)
	}
	for i := range s.ServiceAccounts {
		dropManifestCopies(s.ServiceAccounts[i].Annotations)
	}
}

func dropManifestCopies(annotations map[string]string) {
	for key, value := range annotations {
		if isManifestCopy(key, value) {
			delete(annotations, key)
		}
	}
}
