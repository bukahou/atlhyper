// sdk/k8s/provider.go
// K8s Provider 主体实现
package k8s

import (
	"context"
	"fmt"
	"log"

	"AtlHyper/atlhyper_agent/sdk"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8sProvider K8s SDK 提供者实现
type K8sProvider struct {
	// 底层客户端
	runtimeClient client.Client
	coreClientset *kubernetes.Clientset
	metricsClient *metricsclient.Clientset
	restConfig    *rest.Config

	// 子组件
	podOperator        *k8sPodOperator
	nodeOperator       *k8sNodeOperator
	deploymentOperator *k8sDeploymentOperator
	metricsProvider    *k8sMetricsProvider
	clusterInfo        *k8sClusterInfo
}

// NewK8sProvider 创建 K8s 提供者
func NewK8sProvider(cfg sdk.ProviderConfig) (sdk.SDKProvider, error) {
	var err error
	var restConfig *rest.Config

	// 1. 加载配置
	if cfg.Kubeconfig != "" {
		restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
		if err != nil {
			log.Printf("[sdk/k8s] kubeconfig 加载失败: %v, 尝试 InCluster", err)
			restConfig = nil
		} else {
			log.Printf("[sdk/k8s] 使用 kubeconfig: %s", cfg.Kubeconfig)
		}
	}

	// 2. 若 kubeconfig 失败，尝试 InCluster 模式
	if restConfig == nil {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("获取 K8s 配置失败: %w", err)
		}
		log.Println("[sdk/k8s] 使用 InCluster 配置")
	}

	// 3. 初始化各类客户端
	runtimeClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("初始化 runtime client 失败: %w", err)
	}

	coreClientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 core clientset 失败: %w", err)
	}

	// metrics client 可选，失败不影响
	metricsClient, err := metricsclient.NewForConfig(restConfig)
	if err != nil {
		log.Printf("[sdk/k8s] metrics client 初始化失败 (非致命): %v", err)
		metricsClient = nil
	}

	p := &K8sProvider{
		runtimeClient: runtimeClient,
		coreClientset: coreClientset,
		metricsClient: metricsClient,
		restConfig:    restConfig,
	}

	// 4. 初始化子组件
	p.podOperator = newK8sPodOperator(coreClientset)
	p.nodeOperator = newK8sNodeOperator(coreClientset, runtimeClient)
	p.deploymentOperator = newK8sDeploymentOperator(coreClientset)
	p.metricsProvider = newK8sMetricsProvider(metricsClient)
	p.clusterInfo = newK8sClusterInfo(coreClientset, restConfig)

	log.Println("[sdk/k8s] 初始化成功")
	return p, nil
}

// ==================== 资源操作接口 ====================

func (p *K8sProvider) Pods() sdk.PodOperator {
	return p.podOperator
}

func (p *K8sProvider) Nodes() sdk.NodeOperator {
	return p.nodeOperator
}

func (p *K8sProvider) Deployments() sdk.DeploymentOperator {
	return p.deploymentOperator
}

func (p *K8sProvider) Metrics() sdk.MetricsProvider {
	return p.metricsProvider
}

func (p *K8sProvider) Cluster() sdk.ClusterInfo {
	return p.clusterInfo
}

// ==================== 底层客户端访问 ====================

func (p *K8sProvider) RuntimeClient() client.Client {
	return p.runtimeClient
}

func (p *K8sProvider) CoreClient() *kubernetes.Clientset {
	return p.coreClientset
}

func (p *K8sProvider) RestConfig() *rest.Config {
	return p.restConfig
}

func (p *K8sProvider) MetricsClient() *metricsclient.Clientset {
	return p.metricsClient
}

func (p *K8sProvider) HasMetricsServer() bool {
	return p.metricsClient != nil
}

// ==================== 生命周期 ====================

func (p *K8sProvider) Close() error {
	// K8s 客户端通常不需要显式关闭
	return nil
}

// ==================== ResourceLister 实现 ====================

func (p *K8sProvider) ListPods(ctx context.Context, namespace string) ([]sdk.PodInfo, error) {
	return listPods(ctx, p.coreClientset, namespace)
}

func (p *K8sProvider) ListNodes(ctx context.Context) ([]sdk.NodeInfo, error) {
	return listNodes(ctx, p.coreClientset)
}

func (p *K8sProvider) ListDeployments(ctx context.Context, namespace string) ([]sdk.DeploymentInfo, error) {
	return listDeployments(ctx, p.coreClientset, namespace)
}

func (p *K8sProvider) ListServices(ctx context.Context, namespace string) ([]sdk.ServiceInfo, error) {
	return listServices(ctx, p.coreClientset, namespace)
}

func (p *K8sProvider) ListNamespaces(ctx context.Context) ([]sdk.NamespaceInfo, error) {
	return listNamespaces(ctx, p.coreClientset)
}

func (p *K8sProvider) ListConfigMaps(ctx context.Context, namespace string) ([]sdk.ConfigMapInfo, error) {
	return listConfigMaps(ctx, p.coreClientset, namespace)
}

func (p *K8sProvider) ListIngresses(ctx context.Context, namespace string) ([]sdk.IngressInfo, error) {
	return listIngresses(ctx, p.coreClientset, namespace)
}

// ==================== ResourceGetter 实现 ====================

func (p *K8sProvider) GetPod(ctx context.Context, key sdk.ObjectKey) (*sdk.PodInfo, error) {
	return getPod(ctx, p.coreClientset, key)
}

func (p *K8sProvider) GetNode(ctx context.Context, name string) (*sdk.NodeInfo, error) {
	return getNode(ctx, p.coreClientset, name)
}

func (p *K8sProvider) GetDeployment(ctx context.Context, key sdk.ObjectKey) (*sdk.DeploymentInfo, error) {
	return getDeployment(ctx, p.coreClientset, key)
}

func (p *K8sProvider) GetNamespace(ctx context.Context, name string) (*sdk.NamespaceInfo, error) {
	return getNamespace(ctx, p.coreClientset, name)
}
