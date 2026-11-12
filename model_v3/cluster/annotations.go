package cluster

// manifestCopyAnnotations 是各类部署工具写在资源上的「整份清单副本」。
//
// 这类注解把资源的完整清单再存一遍：体积大，内容与快照里的结构化字段完全重复，
// 展示与分析都只使用结构化字段。因此不随快照上报，Master 收到快照时也不保留。
var manifestCopyAnnotations = map[string]struct{}{
	"kubectl.kubernetes.io/last-applied-configuration": {}, // kubectl apply
	"kapp.k14s.io/original":                            {}, // Carvel kapp
	"objectset.rio.cattle.io/applied":                  {}, // Rancher wrangler
	"banzaicloud.com/last-applied":                     {}, // k8s-objectmatcher
}

// ReportableAnnotations 返回可以随快照上报的注解：去掉整份清单副本，其余原样保留。
//
// 不修改传入的 map；没有需要去掉的键时直接返回原 map，全部被去掉时返回 nil。
func ReportableAnnotations(src map[string]string) map[string]string {
	dropped := 0
	for key := range src {
		if _, ok := manifestCopyAnnotations[key]; ok {
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
		if _, ok := manifestCopyAnnotations[key]; !ok {
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
	for key := range manifestCopyAnnotations {
		delete(annotations, key)
	}
}
