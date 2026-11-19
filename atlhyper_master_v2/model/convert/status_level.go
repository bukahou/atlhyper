package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/cluster"
)

// failingWaitReasons 容器卡在等待、且不会自己好起来的原因（K8s 的原因代码）
var failingWaitReasons = map[string]bool{
	"CrashLoopBackOff":           true,
	"ImagePullBackOff":           true,
	"ErrImagePull":               true,
	"InvalidImageName":           true,
	"CreateContainerConfigError": true,
	"CreateContainerError":       true,
	"RunContainerError":          true,
}

// containerStateLevel 容器状态 → 显示等级
func containerStateLevel(c *cluster.PodContainerDetail) model.StatusLevel {
	switch c.State {
	case "running":
		if c.Ready {
			return model.StatusSuccess
		}
		return model.StatusWarning
	case "waiting":
		if failingWaitReasons[c.StateReason] {
			return model.StatusError
		}
		return model.StatusWarning
	case "terminated":
		if c.StateReason == "Completed" {
			return model.StatusInfo
		}
		return model.StatusError
	default:
		return model.StatusDefault
	}
}

// failingContainerReason 第一个处于故障状态的容器的原因代码；没有为空
func failingContainerReason(p *cluster.Pod) string {
	for i := range p.Containers {
		if containerStateLevel(&p.Containers[i]) == model.StatusError {
			return p.Containers[i].StateReason
		}
	}
	return ""
}

// podStatusLevel Pod → 显示等级。阶段是 Running 的 Pod 也可能有容器在 CrashLoopBackOff，要看容器
func podStatusLevel(p *cluster.Pod) model.StatusLevel {
	switch p.Status.Phase {
	case "Failed":
		return model.StatusError
	case "Succeeded":
		return model.StatusInfo
	case "Pending", "Running":
		if failingContainerReason(p) != "" {
			return model.StatusError
		}
		if p.Status.Phase == "Pending" || !p.IsReady() {
			return model.StatusWarning
		}
		return model.StatusSuccess
	default:
		return model.StatusDefault
	}
}

// namespaceStatusLevel 命名空间阶段 → 显示等级
func namespaceStatusLevel(phase string) model.StatusLevel {
	switch phase {
	case "Active":
		return model.StatusSuccess
	case "Terminating":
		return model.StatusInfo
	default:
		return model.StatusDefault
	}
}
