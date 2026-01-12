// gateway/handler/api/ops/handler.go
// 操作命令处理器
package ops

import (
	"fmt"
	"time"

	"AtlHyper/atlhyper_master/gateway/handler/control"
	"AtlHyper/atlhyper_master/gateway/middleware/response"

	"github.com/gin-gonic/gin"
)

// HandleGetPodLogs 获取 Pod 日志
// POST /uiapi/ops/pod/logs
func HandleGetPodLogs(c *gin.Context) {
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace"  binding:"required"`
		Pod       string `json:"pod"        binding:"required"`
		Container string `json:"container"`      // 可选
		TailLines int    `json:"tailLines"`      // 可选：默认 50
		Timeout   int    `json:"timeoutSeconds"` // 可选：默认 20
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "bad request: "+err.Error())
		return
	}
	if req.TailLines <= 0 {
		req.TailLines = 50
	}
	if req.Timeout <= 0 {
		req.Timeout = 20
	}

	cmd := control.Command{
		ID:   control.GenID(),
		Type: "PodGetLogs",
		Target: map[string]string{
			"ns":  req.Namespace,
			"pod": req.Pod,
		},
		Args: map[string]any{
			"tailLines": req.TailLines,
		},
		Idem: control.Idem("getlogs", req.ClusterID, req.Namespace, req.Pod, req.TailLines, req.Container),
		Op:   "add",
	}
	if req.Container != "" {
		cmd.Args["container"] = req.Container
	}

	// 1) 入队
	control.UpsertCommand(req.ClusterID, cmd)

	// 2) 同步等待 ACK
	ack, ok := control.WaitAck(req.ClusterID, cmd.ID, time.Duration(req.Timeout)*time.Second)
	if !ok {
		response.Error(c, fmt.Sprintf("timeout waiting logs for command %s", cmd.ID))
		return
	}

	// 3) 返回日志
	response.Success(c, "ok", gin.H{
		"commandID": cmd.ID,
		"status":    ack.Status,
		"logs":      ack.Message,
		"errorCode": ack.ErrorCode,
	})
}

// HandleRestartPod 重启 Pod
// POST /uiapi/ops/pod/restart
func HandleRestartPod(c *gin.Context) {
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace"  binding:"required"`
		Pod       string `json:"pod"        binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "bad request: "+err.Error())
		return
	}

	cmd := control.Command{
		ID:   control.GenID(),
		Type: "PodRestart",
		Target: map[string]string{
			"ns":  req.Namespace,
			"pod": req.Pod,
		},
		Idem: control.Idem("restart", req.ClusterID, req.Namespace, req.Pod),
		Op:   "add",
	}

	control.UpsertCommand(req.ClusterID, cmd)

	response.Success(c, "command enqueued", gin.H{
		"commandID": cmd.ID,
		"type":      cmd.Type,
		"target":    cmd.Target,
	})
}

// HandleCordonNode 封锁节点
// POST /uiapi/ops/node/cordon
func HandleCordonNode(c *gin.Context) {
	handleNodeCordon(c, true)
}

// HandleUncordonNode 解封节点
// POST /uiapi/ops/node/uncordon
func HandleUncordonNode(c *gin.Context) {
	handleNodeCordon(c, false)
}

func handleNodeCordon(c *gin.Context, cordon bool) {
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Node      string `json:"node"      binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "bad request: "+err.Error())
		return
	}

	cmdType := "NodeCordon"
	if !cordon {
		cmdType = "NodeUncordon"
	}

	cmd := control.Command{
		ID:   control.GenID(),
		Type: cmdType,
		Target: map[string]string{
			"node": req.Node,
		},
		Args: nil,
		Idem: control.Idem(cmdType, req.ClusterID, req.Node),
		Op:   "add",
	}

	control.UpsertCommand(req.ClusterID, cmd)
	response.Success(c, "command enqueued", gin.H{
		"commandID": cmd.ID,
		"type":      cmd.Type,
		"target":    cmd.Target,
	})
}

// HandleScaleWorkload 修改副本数
// POST /uiapi/ops/workload/scale
func HandleScaleWorkload(c *gin.Context) {
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace"  binding:"required"`
		Kind      string `json:"kind"`
		Name      string `json:"name"       binding:"required"`
		Replicas  int    `json:"replicas"   binding:"required"`
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

	cmd := control.Command{
		ID:   control.GenID(),
		Type: "ScaleWorkload",
		Target: map[string]string{
			"ns":   req.Namespace,
			"kind": req.Kind,
			"name": req.Name,
		},
		Args: map[string]any{
			"replicas": req.Replicas,
		},
		Idem: control.Idem("scale", req.ClusterID, req.Namespace, req.Kind, req.Name, req.Replicas),
		Op:   "add",
	}

	control.UpsertCommand(req.ClusterID, cmd)
	response.Success(c, "command enqueued", gin.H{
		"commandID": cmd.ID,
		"type":      cmd.Type,
		"target":    cmd.Target,
		"args":      cmd.Args,
	})
}

// HandleUpdateImage 更新镜像
// POST /uiapi/ops/workload/updateImage
func HandleUpdateImage(c *gin.Context) {
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace"  binding:"required"`
		Kind      string `json:"kind"`
		Name      string `json:"name"       binding:"required"`
		NewImage  string `json:"newImage"   binding:"required"`
		OldImage  string `json:"oldImage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "bad request: "+err.Error())
		return
	}
	if req.Kind == "" {
		req.Kind = "Deployment"
	}

	args := map[string]any{
		"newImage": req.NewImage,
	}
	if req.OldImage != "" {
		args["_preconditions"] = map[string]any{
			"currentImage": req.OldImage,
		}
	}

	cmd := control.Command{
		ID:   control.GenID(),
		Type: "UpdateImage",
		Target: map[string]string{
			"ns":   req.Namespace,
			"kind": req.Kind,
			"name": req.Name,
		},
		Args: args,
		Idem: control.Idem("updimg", req.ClusterID, req.Namespace, req.Kind, req.Name, req.NewImage, req.OldImage),
		Op:   "add",
	}

	control.UpsertCommand(req.ClusterID, cmd)
	response.Success(c, "command enqueued", gin.H{
		"commandID": cmd.ID,
		"type":      cmd.Type,
		"target":    cmd.Target,
		"args":      cmd.Args,
	})
}
