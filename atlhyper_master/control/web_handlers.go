// control/web_handlers.go
package control

import (
	response "AtlHyper/atlhyper_master/server/api/response"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

//
// ============================= 工具函数（命令ID / 幂等键） =============================
//
// genID：生成唯一命令ID（用于审计、ACK回执关联）。
// 说明：ID 只是“这次下发”的唯一性标识；真正避免重复执行靠 Idem（幂等键）。
func genID() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("cmd-%d", timeNowUnixNano())))
	return "cmd-" + hex.EncodeToString(sum[:8])
}

// timeNowUnixNano：可替换的时间函数（便于单元测试打桩）。
var timeNowUnixNano = func() int64 { return time.Now().UnixNano() }

// idem：计算幂等键（避免重复执行）。
// 典型输入：动作 + 集群ID + 资源定位 + 关键参数（如 replicas / newImage）。
func idem(parts ...any) string {
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprint(h, "|", p)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

//
// ============================= 路由注册 =============================
//
// RegisterWebOpsRoutes —— 面向“Web/管理端”的操作入口。
// 前缀：/ingest/ops/admin/*
// 说明：这些接口负责把“值令（Command）”塞入指定集群的 CommandSet，
//       由 Agent 通过 /ops/watch 拉取执行。建议挂鉴权/审计中间件。
func RegisterWebOpsRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/ops/admin")

	// 1) 重启 Pod（删除指定 Pod，通常会被控制器拉起新的副本）
	admin.POST("/pod/restart", HandleWebRestartPod)

	// 2) 封锁 / 解锁 Node（仅设置 unschedulable，不做 drain）
	admin.POST("/node/cordon", HandleWebCordonNode)
	admin.POST("/node/uncordon", HandleWebUncordonNode)

	// 3) 更新镜像（指定工作负载到某个“完整镜像”）
	admin.POST("/workload/updateImage", HandleWebUpdateImage)

	// 4) 修改副本数（Deployment/StatefulSet，默认 Deployment）
	admin.POST("/workload/scale", HandleWebScaleWorkload)
}

//
// ============================= 1) 重启 Pod =============================
//
// POST /ingest/ops/admin/pod/restart
// 请求体：{ "clusterID": "...", "namespace": "default", "pod": "xxx-123" }
// 语义：删除该 Pod（支持幂等：若已被替换或不存在则 Skipped）
// Agent 侧建议：调用 K8s Delete Pod（带默认 gracePeriodSeconds=10~30）
func HandleWebRestartPod(c *gin.Context) {
	// 解析请求参数（必填：clusterID/ns/pod）
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace"  binding:"required"`
		Pod       string `json:"pod"        binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "bad request: "+err.Error())
		return
	}

	// 构造命令（不传 Args：简单直接；若需优雅期可在 Args 增加 gracePeriodSeconds）
	cmd := Command{
		ID:   genID(),
		Type: "PodRestart",
		Target: map[string]string{
			"ns":  req.Namespace,
			"pod": req.Pod,
		},
		Idem: idem("restart", req.ClusterID, req.Namespace, req.Pod), // 幂等：动作+集群+ns+pod
		Op:   "add",
	}

	// 入队并唤醒等待的 Agent
	upsertCommand(req.ClusterID, cmd)

	// 返回统一响应（前端可直接显示 Target 作为回显）
	response.Success(c, "command enqueued", gin.H{
		"commandID": cmd.ID,
		"type":      cmd.Type,
		"target":    cmd.Target,
	})
}

//
// ============================= 2) 仅封锁/解封 Node =============================
//
// 只设置/取消 node.spec.unschedulable；不做 drain。
// - 封锁：POST /ingest/ops/admin/node/cordon
// - 解封：POST /ingest/ops/admin/node/uncordon
// 请求体：{ "clusterID": "...", "node": "node-1" }
// 幂等：重复封锁/解封同一状态将被 Agent 侧视为 Skipped。
func HandleWebCordonNode(c *gin.Context)   { handleWebNodeCordonSimple(c, true) }
func HandleWebUncordonNode(c *gin.Context) { handleWebNodeCordonSimple(c, false) }

func handleWebNodeCordonSimple(c *gin.Context, cordon bool) {
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"` // 目标集群
		Node      string `json:"node"      binding:"required"` // 节点名
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "bad request: "+err.Error())
		return
	}

	cmdType := "NodeCordon"
	if !cordon {
		cmdType = "NodeUncordon"
	}

	cmd := Command{
		ID:   genID(),
		Type: cmdType,
		Target: map[string]string{
			"node": req.Node,
		},
		Args: nil,                                              // 不做 drain，无需 Args
		Idem: idem(cmdType, req.ClusterID, req.Node),           // 幂等：动作+集群+node
		Op:   "add",
	}

	upsertCommand(req.ClusterID, cmd)
	response.Success(c, "command enqueued", gin.H{
		"commandID": cmd.ID,
		"type":      cmd.Type,
		"target":    cmd.Target,
	})
}

//
// ============================= 3) 更新镜像 =============================
//
// POST /ingest/ops/admin/workload/updateImage
// 请求体（精简版）：
// {
//   "clusterID": "...",          // 必填：集群
//   "namespace": "default",      // 必填：命名空间
//   "name": "web-api",           // 必填：workload 名
//   "newImage": "repo/app:v1",   // 必填：目标完整镜像（含仓库/标签或digest）
//   "kind": "Deployment",        // 可选：默认 "Deployment"，也可 "StatefulSet"
//   "oldImage": "repo/app:v0.9"  // 可选：CAS 保护（不匹配则跳过/失败，策略由 Agent 定）
// }
// 说明：未指定 container——Agent 侧需做判断：单容器直接更新；多容器建议返回失败（避免误改）。
func HandleWebUpdateImage(c *gin.Context) {
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace"  binding:"required"`
		Kind      string `json:"kind"`                          // 默认 Deployment
		Name      string `json:"name"       binding:"required"` // workload 名
		NewImage  string `json:"newImage"   binding:"required"` // 目标完整镜像
		OldImage  string `json:"oldImage"`                      // 可选：CAS 保护
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "bad request: "+err.Error())
		return
	}
	if req.Kind == "" {
		req.Kind = "Deployment"
	}

	// Args：仅携带新镜像；若传了 oldImage，则下发为前置条件（由 Agent 执行 CAS 校验）
	args := map[string]any{
		"newImage": req.NewImage,
	}
	if req.OldImage != "" {
		args["_preconditions"] = map[string]any{
			"currentImage": req.OldImage,
		}
	}

	cmd := Command{
		ID:   genID(),
		Type: "UpdateImage",
		Target: map[string]string{
			"ns":   req.Namespace,
			"kind": req.Kind,
			"name": req.Name,
		},
		Args: args,
		// 幂等键：动作 + 集群 + 资源定位 + 目标镜像（+可选旧镜像）
		Idem: idem("updimg", req.ClusterID, req.Namespace, req.Kind, req.Name, req.NewImage, req.OldImage),
		Op:   "add",
	}

	upsertCommand(req.ClusterID, cmd)
	response.Success(c, "command enqueued", gin.H{
		"commandID": cmd.ID,
		"type":      cmd.Type,
		"target":    cmd.Target,
		"args":      cmd.Args,
	})
}

//
// ============================= 4) 修改副本数 =============================
//
// POST /ingest/ops/admin/workload/scale
// 请求体：
// {
//   "clusterID": "...",          // 必填：集群
//   "namespace": "default",      // 必填：命名空间
//   "name": "web-api",           // 必填：workload 名
//   "replicas": 3,               // 必填：目标副本数（>=0）
//   "kind": "Deployment"         // 可选：默认 "Deployment"，也可 "StatefulSet"
// }
// 幂等：若当前副本数已等于目标，Agent 侧直接返回 Skipped。
func HandleWebScaleWorkload(c *gin.Context) {
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace"  binding:"required"`
		Kind      string `json:"kind"`                          // 默认 Deployment
		Name      string `json:"name"       binding:"required"`
		Replicas  int    `json:"replicas"   binding:"required"` // 目标副本数（>=0）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "bad request: "+err.Error())
		return
	}
	if req.Kind == "" {
		req.Kind = "Deployment"
	}
	if req.Replicas < 0 {
		response.Error(c, "replicas must be >= 0")
		return
	}

	cmd := Command{
		ID:   genID(),
		Type: "ScaleWorkload",
		Target: map[string]string{
			"ns":   req.Namespace,
			"kind": req.Kind,
			"name": req.Name,
		},
		Args: map[string]any{
			"replicas": req.Replicas,
		},
		// 幂等键：动作 + 集群 + 资源定位 + 目标副本数
		Idem: idem("scale", req.ClusterID, req.Namespace, req.Kind, req.Name, req.Replicas),
		Op:   "add",
	}

	upsertCommand(req.ClusterID, cmd)
	response.Success(c, "command enqueued", gin.H{
		"commandID": cmd.ID,
		"type":      cmd.Type,
		"target":    cmd.Target,
		"args":      cmd.Args,
	})
}
