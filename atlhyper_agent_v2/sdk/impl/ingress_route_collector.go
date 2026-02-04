// atlhyper_agent_v2/sdk/impl/ingress_route_collector.go
// Traefik IngressRoute 采集器实现
package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
)

var routeLog = logger.Module("IngressRouteCollector")

// ingressRouteCollector IngressRoute 采集器实现
type ingressRouteCollector struct {
	k8sClient sdk.K8sClient
}

// NewIngressRouteCollector 创建 IngressRoute 采集器
func NewIngressRouteCollector(k8sClient sdk.K8sClient) sdk.IngressRouteCollector {
	return &ingressRouteCollector{
		k8sClient: k8sClient,
	}
}

// CollectRoutes 采集所有 IngressRoute 配置
// 优先采集 Traefik IngressRoute CRD，如果没有则采集标准 Kubernetes Ingress
func (c *ingressRouteCollector) CollectRoutes(ctx context.Context) ([]model.IngressRouteInfo, error) {
	var routes []model.IngressRouteInfo

	// 1. 尝试获取 IngressRoute CRD (traefik.io/v1alpha1)
	resp, err := c.k8sClient.Dynamic(ctx, sdk.DynamicRequest{
		Path: "/apis/traefik.io/v1alpha1/ingressroutes",
	})
	if err != nil {
		// 尝试旧版本 API (traefik.containo.us/v1alpha1)
		resp, err = c.k8sClient.Dynamic(ctx, sdk.DynamicRequest{
			Path: "/apis/traefik.containo.us/v1alpha1/ingressroutes",
		})
	}

	if err == nil {
		// 解析 IngressRoute CRD
		var result ingressRouteList
		if err := json.Unmarshal(resp.Body, &result); err == nil && len(result.Items) > 0 {
			for _, ir := range result.Items {
				parsed := c.parseIngressRoute(ir)
				routes = append(routes, parsed...)
			}
			routeLog.Debug("采集 IngressRoute CRD 完成", "count", len(routes))
			return routes, nil
		}
	}

	// 2. Fallback: 采集标准 Kubernetes Ingress
	routes, err = c.collectStandardIngress(ctx)
	if err != nil {
		routeLog.Debug("采集标准 Ingress 失败", "err", err)
		return nil, nil
	}

	if len(routes) > 0 {
		routeLog.Debug("采集标准 Ingress 完成", "count", len(routes))
	}
	return routes, nil
}

// collectStandardIngress 采集标准 Kubernetes Ingress
func (c *ingressRouteCollector) collectStandardIngress(ctx context.Context) ([]model.IngressRouteInfo, error) {
	var routes []model.IngressRouteInfo

	// 获取所有 Ingress (networking.k8s.io/v1)
	resp, err := c.k8sClient.Dynamic(ctx, sdk.DynamicRequest{
		Path: "/apis/networking.k8s.io/v1/ingresses",
	})
	if err != nil {
		return nil, err
	}

	var result k8sIngressList
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("解析 Ingress 列表失败: %w", err)
	}

	for _, ing := range result.Items {
		parsed := c.parseK8sIngress(ing)
		routes = append(routes, parsed...)
	}

	return routes, nil
}

// parseK8sIngress 解析标准 Kubernetes Ingress
func (c *ingressRouteCollector) parseK8sIngress(ing k8sIngress) []model.IngressRouteInfo {
	var routes []model.IngressRouteInfo

	// 构建 TLS hosts 集合
	tlsHosts := make(map[string]bool)
	for _, tls := range ing.Spec.TLS {
		for _, host := range tls.Hosts {
			tlsHosts[host] = true
		}
	}

	// 解析每个规则
	for _, rule := range ing.Spec.Rules {
		if rule.Host == "" || rule.HTTP == nil {
			continue
		}

		tls := tlsHosts[rule.Host]

		for _, path := range rule.HTTP.Paths {
			pathPrefix := path.Path
			if pathPrefix == "" {
				pathPrefix = "/"
			}

			// 获取后端服务信息
			if path.Backend.Service == nil {
				continue
			}

			serviceName := path.Backend.Service.Name
			servicePort := 80
			if path.Backend.Service.Port.Number > 0 {
				servicePort = path.Backend.Service.Port.Number
			}

			// 构建 Traefik service key 格式
			// Traefik 使用此格式作为 Prometheus metrics 中的 service label
			serviceKey := fmt.Sprintf("%s-%s-%d@kubernetes",
				ing.Metadata.Namespace, serviceName, servicePort)

			routes = append(routes, model.IngressRouteInfo{
				Name:        ing.Metadata.Name,
				Namespace:   ing.Metadata.Namespace,
				Domain:      rule.Host,
				PathPrefix:  pathPrefix,
				ServiceKey:  serviceKey,
				ServiceName: serviceName,
				ServicePort: servicePort,
				TLS:         tls,
			})
		}
	}

	return routes
}

// parseIngressRoute 解析单个 IngressRoute
func (c *ingressRouteCollector) parseIngressRoute(ir ingressRoute) []model.IngressRouteInfo {
	var routes []model.IngressRouteInfo

	// 判断是否启用 TLS
	tls := ir.Spec.TLS != nil

	// 解析每个路由规则
	for _, route := range ir.Spec.Routes {
		// 解析 match 规则
		domain, pathPrefix := parseMatchRule(route.Match)
		if domain == "" {
			continue // 无法解析域名，跳过
		}

		// 处理每个后端 Service
		for _, svc := range route.Services {
			port := svc.Port
			if port == 0 {
				port = 80 // 默认端口
			}

			// 构建 Traefik service key
			// 格式: {namespace}-{service}-{port}@kubernetes
			serviceKey := fmt.Sprintf("%s-%s-%d@kubernetes",
				ir.Metadata.Namespace, svc.Name, port)

			routes = append(routes, model.IngressRouteInfo{
				Name:        ir.Metadata.Name,
				Namespace:   ir.Metadata.Namespace,
				Domain:      domain,
				PathPrefix:  pathPrefix,
				ServiceKey:  serviceKey,
				ServiceName: svc.Name,
				ServicePort: port,
				TLS:         tls,
			})
		}
	}

	return routes
}

// parseMatchRule 解析 Traefik match 规则
// 示例: Host(`example.com`) && PathPrefix(`/api`)
// 返回: domain, pathPrefix
func parseMatchRule(match string) (domain, pathPrefix string) {
	// 默认路径
	pathPrefix = "/"

	// 匹配 Host(`xxx`)
	hostRe := regexp.MustCompile(`Host\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	if matches := hostRe.FindStringSubmatch(match); len(matches) > 1 {
		domain = matches[1]
	}

	// 匹配 PathPrefix(`xxx`)
	pathRe := regexp.MustCompile(`PathPrefix\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	if matches := pathRe.FindStringSubmatch(match); len(matches) > 1 {
		pathPrefix = matches[1]
	}

	// 匹配 Path(`xxx`)
	pathExactRe := regexp.MustCompile(`Path\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	if matches := pathExactRe.FindStringSubmatch(match); len(matches) > 1 {
		pathPrefix = matches[1]
	}

	// 处理 HostRegexp (简单提取第一个域名)
	if domain == "" {
		hostRegexpRe := regexp.MustCompile(`HostRegexp\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
		if matches := hostRegexpRe.FindStringSubmatch(match); len(matches) > 1 {
			// 简单处理: 移除正则特殊字符
			domain = strings.ReplaceAll(matches[1], "{", "")
			domain = strings.ReplaceAll(domain, "}", "")
			domain = strings.ReplaceAll(domain, ".*", "")
			domain = strings.TrimPrefix(domain, "^")
			domain = strings.TrimSuffix(domain, "$")
		}
	}

	return domain, pathPrefix
}

// =============================================================================
// IngressRoute CRD 结构定义 (仅用于 JSON 解析)
// =============================================================================

type ingressRouteList struct {
	Items []ingressRoute `json:"items"`
}

type ingressRoute struct {
	Metadata ingressRouteMetadata `json:"metadata"`
	Spec     ingressRouteSpec     `json:"spec"`
}

type ingressRouteMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ingressRouteSpec struct {
	EntryPoints []string             `json:"entryPoints,omitempty"`
	Routes      []ingressRouteRoute  `json:"routes"`
	TLS         *ingressRouteTLS     `json:"tls,omitempty"`
}

type ingressRouteRoute struct {
	Match    string                   `json:"match"`
	Kind     string                   `json:"kind,omitempty"`
	Services []ingressRouteService    `json:"services,omitempty"`
}

type ingressRouteService struct {
	Name string `json:"name"`
	Port int    `json:"port,omitempty"`
}

type ingressRouteTLS struct {
	SecretName string `json:"secretName,omitempty"`
}

// =============================================================================
// 标准 Kubernetes Ingress 结构定义 (networking.k8s.io/v1)
// =============================================================================

type k8sIngressList struct {
	Items []k8sIngress `json:"items"`
}

type k8sIngress struct {
	Metadata k8sIngressMetadata `json:"metadata"`
	Spec     k8sIngressSpec     `json:"spec"`
}

type k8sIngressMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type k8sIngressSpec struct {
	IngressClassName *string          `json:"ingressClassName,omitempty"`
	TLS              []k8sIngressTLS  `json:"tls,omitempty"`
	Rules            []k8sIngressRule `json:"rules,omitempty"`
}

type k8sIngressTLS struct {
	Hosts      []string `json:"hosts,omitempty"`
	SecretName string   `json:"secretName,omitempty"`
}

type k8sIngressRule struct {
	Host string              `json:"host,omitempty"`
	HTTP *k8sIngressRuleHTTP `json:"http,omitempty"`
}

type k8sIngressRuleHTTP struct {
	Paths []k8sIngressPath `json:"paths,omitempty"`
}

type k8sIngressPath struct {
	Path     string            `json:"path,omitempty"`
	PathType *string           `json:"pathType,omitempty"`
	Backend  k8sIngressBackend `json:"backend"`
}

type k8sIngressBackend struct {
	Service *k8sIngressBackendService `json:"service,omitempty"`
}

type k8sIngressBackendService struct {
	Name string                       `json:"name"`
	Port k8sIngressBackendServicePort `json:"port"`
}

type k8sIngressBackendServicePort struct {
	Number int    `json:"number,omitempty"`
	Name   string `json:"name,omitempty"`
}
