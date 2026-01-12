// sdk/k8s/lister.go
// ResourceLister 和 ResourceGetter K8s 实现
package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent/sdk"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ==================== List 实现 ====================

func listPods(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]sdk.PodInfo, error) {
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	podList, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]sdk.PodInfo, len(podList.Items))
	for i, pod := range podList.Items {
		result[i] = convertPod(&pod)
	}
	return result, nil
}

func listNodes(ctx context.Context, cs *kubernetes.Clientset) ([]sdk.NodeInfo, error) {
	nodeList, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]sdk.NodeInfo, len(nodeList.Items))
	for i, node := range nodeList.Items {
		result[i] = convertNode(&node)
	}
	return result, nil
}

func listDeployments(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]sdk.DeploymentInfo, error) {
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	deployList, err := cs.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]sdk.DeploymentInfo, len(deployList.Items))
	for i, deploy := range deployList.Items {
		result[i] = convertDeployment(&deploy)
	}
	return result, nil
}

func listServices(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]sdk.ServiceInfo, error) {
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	svcList, err := cs.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]sdk.ServiceInfo, len(svcList.Items))
	for i, svc := range svcList.Items {
		result[i] = convertService(&svc)
	}
	return result, nil
}

func listNamespaces(ctx context.Context, cs *kubernetes.Clientset) ([]sdk.NamespaceInfo, error) {
	nsList, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]sdk.NamespaceInfo, len(nsList.Items))
	for i, ns := range nsList.Items {
		result[i] = convertNamespace(&ns)
	}
	return result, nil
}

func listConfigMaps(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]sdk.ConfigMapInfo, error) {
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	cmList, err := cs.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]sdk.ConfigMapInfo, len(cmList.Items))
	for i, cm := range cmList.Items {
		result[i] = convertConfigMap(&cm)
	}
	return result, nil
}

func listIngresses(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]sdk.IngressInfo, error) {
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	ingList, err := cs.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]sdk.IngressInfo, len(ingList.Items))
	for i, ing := range ingList.Items {
		result[i] = convertIngress(&ing)
	}
	return result, nil
}

// ==================== Get 实现 ====================

func getPod(ctx context.Context, cs *kubernetes.Clientset, key sdk.ObjectKey) (*sdk.PodInfo, error) {
	pod, err := cs.CoreV1().Pods(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := convertPod(pod)
	return &info, nil
}

func getNode(ctx context.Context, cs *kubernetes.Clientset, name string) (*sdk.NodeInfo, error) {
	node, err := cs.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := convertNode(node)
	return &info, nil
}

func getDeployment(ctx context.Context, cs *kubernetes.Clientset, key sdk.ObjectKey) (*sdk.DeploymentInfo, error) {
	deploy, err := cs.AppsV1().Deployments(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := convertDeployment(deploy)
	return &info, nil
}

func getNamespace(ctx context.Context, cs *kubernetes.Clientset, name string) (*sdk.NamespaceInfo, error) {
	ns, err := cs.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := convertNamespace(ns)
	return &info, nil
}
