// internal/ops/dispatcher.go
package operations

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	// 根据你的实际目录调整下面三行导入路径
	"AtlHyper/atlhyper_agent/internal/operator/deployment"
	"AtlHyper/atlhyper_agent/internal/operator/node"
	"AtlHyper/atlhyper_agent/internal/operator/pod"
)

// Command 是 Agent 侧执行入口约定的最小字段集合。
// 如果你已经在别处定义了完全一致的结构，可以删掉这里的定义并改为引用你现有的类型。
type Command struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`            // PodRestart / NodeCordon / NodeUncordon / UpdateImage / ScaleWorkload
	Target map[string]string      `json:"target,omitempty"`// 资源定位：ns/pod 或 ns/kind/name 或 node
	Args   map[string]any         `json:"args,omitempty"`  // 参数：newImage/replicas 等
	Idem   string                 `json:"idem,omitempty"`  // 幂等键（透传给上层用于 Ack/去重）
}

// Result 是执行结果的统一返回。
// 上层可直接把它映射为 /ack 的 AckResult；也可在更上一层转换。
type Result struct {
	CommandID string `json:"commandID"`
	Idem      string `json:"idem,omitempty"`
	Status    string `json:"status"`     // Succeeded / Failed / Skipped（当前未细分 NotFound 等，后续可扩展）
	ErrorCode string `json:"errorCode"`  // 预留：Forbidden/NotFound/Conflict...
	Message   string `json:"message"`
}

// Execute 是对外唯一入口：根据命令类型分发到底层具体实现。
func Execute(ctx context.Context, cmd Command) Result {
	switch cmd.Type {
	case "PodRestart":
		ns := cmd.Target["ns"]
		p  := cmd.Target["pod"]
		if ns == "" || p == "" {
			return fail(cmd, "BadRequest", "missing target.ns or target.pod")
		}
		if err := pod.RestartPod(ctx, ns, p); err != nil {
			return fail(cmd, "ExecuteError", fmt.Sprintf("restart pod %s/%s: %v", ns, p, err))
		}
		return ok(cmd, fmt.Sprintf("pod %s/%s deleted (restart requested)", ns, p))

	case "NodeCordon":
		n := cmd.Target["node"]
		if n == "" {
			return fail(cmd, "BadRequest", "missing target.node")
		}
		if err := node.SetNodeSchedulable(n, true); err != nil {
			return fail(cmd, "ExecuteError", fmt.Sprintf("cordon node %s: %v", n, err))
		}
		return ok(cmd, fmt.Sprintf("node %s cordoned", n))

	case "NodeUncordon":
		n := cmd.Target["node"]
		if n == "" {
			return fail(cmd, "BadRequest", "missing target.node")
		}
		if err := node.SetNodeSchedulable(n, false); err != nil {
			return fail(cmd, "ExecuteError", fmt.Sprintf("uncordon node %s: %v", n, err))
		}
		return ok(cmd, fmt.Sprintf("node %s uncordoned", n))

	case "ScaleWorkload":
		ns  := cmd.Target["ns"]
		// kind 目前底层只实现了 Deployment；如果未来加 StatefulSet，这里再扩展分支
		kind := cmd.Target["kind"]
		name := cmd.Target["name"]
		if ns == "" || name == "" {
			return fail(cmd, "BadRequest", "missing target.ns or target.name")
		}
		replicas, err := getReplicas(cmd.Args["replicas"])
		if err != nil {
			return fail(cmd, "BadRequest", "invalid args.replicas")
		}
		// 当前实现只支持 Deployment；如果传了其它 kind，可提前报错或直接忽略
		if kind != "" && kind != "Deployment" {
			return fail(cmd, "BadRequest", "only kind=Deployment is supported for scaling in current agent")
		}
		if err := deployment.UpdateReplicas(ctx, ns, name, replicas); err != nil {
			return fail(cmd, "ExecuteError", fmt.Sprintf("scale %s/%s to %d: %v", ns, name, replicas, err))
		}
		return ok(cmd, fmt.Sprintf("scaled %s/%s to %d", ns, name, replicas))

	case "UpdateImage":
		ns   := cmd.Target["ns"]
		kind := cmd.Target["kind"] // 当前底层实现针对 Deployment
		name := cmd.Target["name"]
		if ns == "" || name == "" {
			return fail(cmd, "BadRequest", "missing target.ns or target.name")
		}
		newImage, _ := getString(cmd.Args["newImage"])
		if newImage == "" {
			return fail(cmd, "BadRequest", "missing args.newImage")
		}
		if kind != "" && kind != "Deployment" {
			return fail(cmd, "BadRequest", "only kind=Deployment is supported for image update in current agent")
		}
		if err := deployment.UpdateAllContainerImages(ctx, ns, name, newImage); err != nil {
			return fail(cmd, "ExecuteError", fmt.Sprintf("update image for %s/%s to %q: %v", ns, name, newImage, err))
		}
		return ok(cmd, fmt.Sprintf("updated image for %s/%s to %q", ns, name, newImage))

    // ===== 新增：获取 Pod 日志 =====
    case "PodGetLogs":
        ns := cmd.Target["ns"]
        p  := cmd.Target["pod"]
        if ns == "" || p == "" {
            return fail(cmd, "BadRequest", "missing target.ns or target.pod")
        }

        // 可选参数：container（字符串），tailLines（数字，默认 50）
        container, _ := getString(cmd.Args["container"])
        tailLines, err := getInt64(cmd.Args["tailLines"])
        if err != nil || tailLines <= 0 {
            tailLines = 50 // 默认 50 行
        }

        logs, err := pod.GetPodLogs(ctx, ns, p, container, tailLines)
        if err != nil {
            return fail(cmd, "ExecuteError", fmt.Sprintf("get logs %s/%s: %v", ns, p, err))
        }

        // 直接把日志放到 Message 里回传（已开启 gzip，通常没问题）
        // 如需更稳妥，可在这里做长度截断或落地到对象存储后回传 URL。
        return ok(cmd, logs)

	default:
		return fail(cmd, "Unsupported", "unknown command type: "+cmd.Type)
	}
}

// ------------------------- 辅助：统一 Result 构造 -------------------------

func ok(cmd Command, msg string) Result {
	return Result{
		CommandID: cmd.ID,
		Idem:      cmd.Idem,
		Status:    "Succeeded",
		Message:   msg,
	}
}

func fail(cmd Command, code, msg string) Result {
	return Result{
		CommandID: cmd.ID,
		Idem:      cmd.Idem,
		Status:    "Failed",
		ErrorCode: code,
		Message:   msg,
	}
}

// ------------------------- 辅助：参数解析与转换 -------------------------

// getString 尝试把 any 转成 string（JSON 反序列化后一般已经是 string）
func getString(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	if s, ok := v.(string); ok {
		return s, true
	}
	return fmt.Sprint(v), true
}

// getReplicas 把 args.replicas 转成 int32，兼容 JSON number(float64)、string、int 等。
func getReplicas(v any) (int32, error) {
	if v == nil {
		return 0, errors.New("replicas is nil")
	}
	switch t := v.(type) {
	case int:
		return int32(t), nil
	case int32:
		return t, nil
	case int64:
		return int32(t), nil
	case float64: // 标准库 JSON 会把数字解到 float64
		return int32(t), nil
	case string:
		i, err := strconv.Atoi(t)
		if err != nil {
			return 0, err
		}
		return int32(i), nil
	default:
		return 0, fmt.Errorf("unsupported replicas type: %T", v)
	}
}

// 新增：把 args.tailLines 解析成 int64
func getInt64(v any) (int64, error) {
    if v == nil {
        return 0, errors.New("nil")
    }
    switch t := v.(type) {
    case int:
        return int64(t), nil
    case int32:
        return int64(t), nil
    case int64:
        return t, nil
    case float64: // JSON number
        return int64(t), nil
    case string:
        i, err := strconv.ParseInt(t, 10, 64)
        if err != nil {
            return 0, err
        }
        return i, nil
    default:
        return 0, fmt.Errorf("unsupported int64 type: %T", v)
    }
}