// internal/readonly/namespace/aggregate_counts.go
package namespace

import (
	"context"

	modelns "AtlHyper/model/namespace"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"AtlHyper/atlhyper_agent/utils"
)

// aggregateCounts —— 全量拉取相关对象，按 NS 聚合计数（Pods/Workloads/Net/Config 等）
// 说明：Pods 由上层一次性传入，避免重复 List；其余对象在此各自全量 List。
func aggregateCounts(ctx context.Context, pods []corev1.Pod) map[string]modelns.NamespaceCounts {
	cs := utils.GetCoreClient()
	out := map[string]modelns.NamespaceCounts{}

	// Pods（用传入的列表）
	for i := range pods {
		p := &pods[i]
		c := out[p.Namespace]
		c.Pods++
		switch p.Status.Phase {
		case corev1.PodRunning:
			c.PodsRunning++
		case corev1.PodPending:
			c.PodsPending++
		case corev1.PodFailed:
			c.PodsFailed++
		case corev1.PodSucceeded:
			c.PodsSucceeded++
		}
		out[p.Namespace] = c
	}

	// 其它对象：全量拉取后逐一加到对应 NS
	addAppsCounts(ctx, cs, out)
	addBatchCounts(ctx, cs, out)
	addServiceCounts(ctx, cs, out)
	addNetCounts(ctx, cs, out)
	addConfigCounts(ctx, cs, out)

	return out
}

func addAppsCounts(ctx context.Context, cs *kubernetes.Clientset, out map[string]modelns.NamespaceCounts) {
	if dps, err := cs.AppsV1().Deployments(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range dps.Items {
			ns := dps.Items[i].Namespace
			c := out[ns]
			c.Deployments++
			out[ns] = c
		}
	}
	if sfs, err := cs.AppsV1().StatefulSets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range sfs.Items {
			ns := sfs.Items[i].Namespace
			c := out[ns]
			c.StatefulSets++
			out[ns] = c
		}
	}
	if dss, err := cs.AppsV1().DaemonSets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range dss.Items {
			ns := dss.Items[i].Namespace
			c := out[ns]
			c.DaemonSets++
			out[ns] = c
		}
	}
}

func addBatchCounts(ctx context.Context, cs *kubernetes.Clientset, out map[string]modelns.NamespaceCounts) {
	if jobs, err := cs.BatchV1().Jobs(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range jobs.Items {
			ns := jobs.Items[i].Namespace
			c := out[ns]
			c.Jobs++
			out[ns] = c
		}
	}
	if cjs, err := cs.BatchV1().CronJobs(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range cjs.Items {
			ns := cjs.Items[i].Namespace
			c := out[ns]
			c.CronJobs++
			out[ns] = c
		}
	}
}

func addServiceCounts(ctx context.Context, cs *kubernetes.Clientset, out map[string]modelns.NamespaceCounts) {
	if svcs, err := cs.CoreV1().Services(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range svcs.Items {
			ns := svcs.Items[i].Namespace
			c := out[ns]
			c.Services++
			out[ns] = c
		}
	}
}

func addNetCounts(ctx context.Context, cs *kubernetes.Clientset, out map[string]modelns.NamespaceCounts) {
	if ings, err := cs.NetworkingV1().Ingresses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range ings.Items {
			ns := ings.Items[i].Namespace
			c := out[ns]
			c.Ingresses++
			out[ns] = c
		}
	}
	if nps, err := cs.NetworkingV1().NetworkPolicies(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range nps.Items {
			ns := nps.Items[i].Namespace
			c := out[ns]
			c.NetworkPolicies++
			out[ns] = c
		}
	}
}

func addConfigCounts(ctx context.Context, cs *kubernetes.Clientset, out map[string]modelns.NamespaceCounts) {
	if cms, err := cs.CoreV1().ConfigMaps(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range cms.Items {
			ns := cms.Items[i].Namespace
			c := out[ns]
			c.ConfigMaps++
			out[ns] = c
		}
	}
	if secs, err := cs.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range secs.Items {
			ns := secs.Items[i].Namespace
			c := out[ns]
			c.Secrets++
			out[ns] = c
		}
	}
	if pvcs, err := cs.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range pvcs.Items {
			ns := pvcs.Items[i].Namespace
			c := out[ns]
			c.PVCs++
			out[ns] = c
		}
	}
	if sas, err := cs.CoreV1().ServiceAccounts(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range sas.Items {
			ns := sas.Items[i].Namespace
			c := out[ns]
			c.ServiceAccounts++
			out[ns] = c
		}
	}
}
