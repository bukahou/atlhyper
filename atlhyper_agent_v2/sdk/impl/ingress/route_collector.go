// Package ingress Ingress Controller 客户端实现
//
// route_collector.go - IngressRoute CRD 采集
//
// 采集 Traefik IngressRoute CRD 或标准 K8s Ingress，
// 建立 Traefik service 名称与实际域名/路径的映射关系。
//
// 优先级:
//  1. traefik.io/v1alpha1/ingressroutes (新版 Traefik CRD)
//  2. traefik.containo.us/v1alpha1/ingressroutes (旧版 Traefik CRD)
//  3. networking.k8s.io/v1/ingresses (标准 K8s Ingress)
package ingress

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// CollectRoutes 采集 IngressRoute / Ingress 配置
func (c *client) CollectRoutes(ctx context.Context) ([]sdk.IngressRouteInfo, error) {
	var routes []sdk.IngressRouteInfo

	// 1. 尝试获取 IngressRoute CRD (traefik.io/v1alpha1)
	resp, err := c.k8sClient.Dynamic(ctx, sdk.DynamicRequest{
		Path: "/apis/traefik.io/v1alpha1/ingressroutes",
	})
	if err != nil {
		// 尝试旧版本 API
		resp, err = c.k8sClient.Dynamic(ctx, sdk.DynamicRequest{
			Path: "/apis/traefik.containo.us/v1alpha1/ingressroutes",
		})
	}

	if err == nil {
		var result ingressRouteList
		if err := json.Unmarshal(resp.Body, &result); err == nil && len(result.Items) > 0 {
			for _, ir := range result.Items {
				parsed := parseIngressRoute(ir)
				routes = append(routes, parsed...)
			}
			log.Debug("采集 IngressRoute CRD 完成", "count", len(routes))
			return routes, nil
		}
	}

	// 2. Fallback: 标准 K8s Ingress
	routes, err = c.collectStandardIngress(ctx)
	if err != nil {
		log.Debug("采集标准 Ingress 失败", "err", err)
		return nil, nil
	}

	if len(routes) > 0 {
		log.Debug("采集标准 Ingress 完成", "count", len(routes))
	}
	return routes, nil
}

// collectStandardIngress 采集标准 K8s Ingress
func (c *client) collectStandardIngress(ctx context.Context) ([]sdk.IngressRouteInfo, error) {
	var routes []sdk.IngressRouteInfo

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
		parsed := parseK8sIngress(ing)
		routes = append(routes, parsed...)
	}

	return routes, nil
}

// =============================================================================
// IngressRoute CRD 解析
// =============================================================================

func parseIngressRoute(ir ingressRoute) []sdk.IngressRouteInfo {
	var routes []sdk.IngressRouteInfo

	tls := ir.Spec.TLS != nil

	for _, route := range ir.Spec.Routes {
		domain, pathPrefix := parseMatchRule(route.Match)
		if domain == "" {
			continue
		}

		for _, svc := range route.Services {
			port := svc.Port
			if port == 0 {
				port = 80
			}

			serviceKey := fmt.Sprintf("%s-%s-%d",
				ir.Metadata.Namespace, svc.Name, port)

			routes = append(routes, sdk.IngressRouteInfo{
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

// parseK8sIngress 解析标准 K8s Ingress
func parseK8sIngress(ing k8sIngress) []sdk.IngressRouteInfo {
	var routes []sdk.IngressRouteInfo

	tlsHosts := make(map[string]bool)
	for _, tls := range ing.Spec.TLS {
		for _, host := range tls.Hosts {
			tlsHosts[host] = true
		}
	}

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

			if path.Backend.Service == nil {
				continue
			}

			serviceName := path.Backend.Service.Name
			servicePort := 80
			if path.Backend.Service.Port.Number > 0 {
				servicePort = path.Backend.Service.Port.Number
			}

			serviceKey := fmt.Sprintf("%s-%s-%d",
				ing.Metadata.Namespace, serviceName, servicePort)

			routes = append(routes, sdk.IngressRouteInfo{
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

// parseMatchRule 解析 Traefik match 规则
// 示例: Host(`example.com`) && PathPrefix(`/api`)
func parseMatchRule(match string) (domain, pathPrefix string) {
	pathPrefix = "/"

	hostRe := regexp.MustCompile(`Host\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	if matches := hostRe.FindStringSubmatch(match); len(matches) > 1 {
		domain = matches[1]
	}

	pathRe := regexp.MustCompile(`PathPrefix\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	if matches := pathRe.FindStringSubmatch(match); len(matches) > 1 {
		pathPrefix = matches[1]
	}

	pathExactRe := regexp.MustCompile(`Path\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	if matches := pathExactRe.FindStringSubmatch(match); len(matches) > 1 {
		pathPrefix = matches[1]
	}

	if domain == "" {
		hostRegexpRe := regexp.MustCompile(`HostRegexp\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
		if matches := hostRegexpRe.FindStringSubmatch(match); len(matches) > 1 {
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
// JSON 结构定义 (仅用于解析)
// =============================================================================

type ingressRouteList struct {
	Items []ingressRoute `json:"items"`
}

type ingressRoute struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		EntryPoints []string `json:"entryPoints,omitempty"`
		Routes      []struct {
			Match    string `json:"match"`
			Services []struct {
				Name string `json:"name"`
				Port int    `json:"port,omitempty"`
			} `json:"services,omitempty"`
		} `json:"routes"`
		TLS *struct {
			SecretName string `json:"secretName,omitempty"`
		} `json:"tls,omitempty"`
	} `json:"spec"`
}

type k8sIngressList struct {
	Items []k8sIngress `json:"items"`
}

type k8sIngress struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		TLS []struct {
			Hosts      []string `json:"hosts,omitempty"`
			SecretName string   `json:"secretName,omitempty"`
		} `json:"tls,omitempty"`
		Rules []struct {
			Host string `json:"host,omitempty"`
			HTTP *struct {
				Paths []struct {
					Path    string `json:"path,omitempty"`
					Backend struct {
						Service *struct {
							Name string `json:"name"`
							Port struct {
								Number int    `json:"number,omitempty"`
								Name   string `json:"name,omitempty"`
							} `json:"port"`
						} `json:"service,omitempty"`
					} `json:"backend"`
				} `json:"paths,omitempty"`
			} `json:"http,omitempty"`
		} `json:"rules,omitempty"`
	} `json:"spec"`
}
