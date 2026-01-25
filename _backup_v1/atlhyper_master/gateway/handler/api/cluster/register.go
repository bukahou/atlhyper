// gateway/handler/api/cluster/register.go
// 集群资源页路由注册
package cluster

import (
	"AtlHyper/atlhyper_master/gateway/middleware/auth"

	"github.com/gin-gonic/gin"
)

// Register 注册集群资源相关路由
// 权限说明：
//   - 公开查看：Pod/Node/Deployment/Service/Namespace/Ingress
//   - Operator：ConfigMap（敏感信息）
func Register(router *gin.RouterGroup) {
	g := router.Group("/cluster")
	{
		// ==================== 公开查看 ====================

		// Pod
		g.POST("/pod/list", GetPodOverviewHandler)
		g.POST("/pod/detail", GetPodDetailHandler)

		// Node
		g.POST("/node/list", GetNodeOverviewHandler)
		g.POST("/node/detail", GetNodeDetailHandler)

		// Deployment
		g.POST("/deployment/list", GetDeploymentOverviewHandler)
		g.POST("/deployment/detail", GetDeploymentDetailHandler)

		// Service
		g.POST("/service/list", GetServiceOverviewHandler)
		g.POST("/service/detail", GetServiceDetailHandler)

		// Namespace
		g.POST("/namespace/list", GetNamespaceOverviewHandler)
		g.POST("/namespace/detail", GetNamespaceDetailHandler)

		// Ingress
		g.POST("/ingress/list", GetIngressOverviewHandler)
		g.POST("/ingress/detail", GetIngressDetailHandler)

		// ==================== 敏感信息（需 Operator 权限） ====================

		sensitive := g.Group("")
		sensitive.Use(auth.RequireAuth(), auth.RequireMinRole(auth.RoleOperator))
		{
			// ConfigMap（可能包含敏感配置）
			sensitive.POST("/configmap/detail", GetConfigMapDetailHandler)
		}
	}
}
