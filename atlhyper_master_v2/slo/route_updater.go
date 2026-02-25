// Package slo SLO 路由映射更新器
//
// route_updater.go - 从 ClusterSnapshot 的 Ingress 资源自动构建路由映射
//
// 解析 K8s Ingress 的 Host + Backend Service，生成 ServiceKey → Domain 映射，
// 写入 slo_route_mapping 表，供 DomainsV2 API 按真实域名分组展示。
package slo

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var routeLog = logger.Module("SLORoute")

// RouteUpdater SLO 路由映射更新器
type RouteUpdater struct {
	sloRepo database.SLORepository
}

// NewRouteUpdater 创建路由映射更新器
func NewRouteUpdater(sloRepo database.SLORepository) *RouteUpdater {
	return &RouteUpdater{sloRepo: sloRepo}
}

// Sync 同步指定集群的路由映射
// 从 ClusterSnapshot.Ingresses 提取 Host → ServiceKey 映射
func (u *RouteUpdater) Sync(store datahub.Store, clusterID string) error {
	snapshot, err := store.GetSnapshot(clusterID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return nil
	}

	u.syncFromIngresses(context.Background(), clusterID, snapshot)
	return nil
}

// syncFromIngresses 从 K8s Ingress 资源构建路由映射
// Ingress Rule: Host + Path → Backend Service (name + port)
// ServiceKey 格式: "{namespace}-{serviceName}-{port}@kubernetes"（与 Traefik metrics 一致）
func (u *RouteUpdater) syncFromIngresses(ctx context.Context, clusterID string, snapshot *model_v2.ClusterSnapshot) {
	now := time.Now()
	count := 0

	for _, ing := range snapshot.Ingresses {
		ns := ing.Summary.Namespace
		ingressName := ing.Summary.Name
		tls := ing.Summary.TLSEnabled

		for _, rule := range ing.Spec.Rules {
			if rule.Host == "" {
				continue
			}
			for _, path := range rule.Paths {
				if path.Backend == nil || path.Backend.Service == nil {
					continue
				}
				svc := path.Backend.Service
				port := int(svc.PortNumber)
				serviceKey := fmt.Sprintf("%s-%s-%d@kubernetes", ns, svc.Name, port)

				if err := u.sloRepo.UpsertRouteMapping(ctx, &database.SLORouteMapping{
					ClusterID:   clusterID,
					Domain:      rule.Host,
					PathPrefix:  path.Path,
					IngressName: ingressName,
					Namespace:   ns,
					TLS:         tls,
					ServiceKey:  serviceKey,
					ServiceName: svc.Name,
					ServicePort: port,
					CreatedAt:   now,
					UpdatedAt:   now,
				}); err != nil {
					routeLog.Error("更新路由映射失败", "cluster", clusterID, "domain", rule.Host, "err", err)
				} else {
					count++
				}
			}
		}
	}

	if count > 0 {
		routeLog.Debug("路由映射同步完成", "cluster", clusterID, "count", count)
	}
}
