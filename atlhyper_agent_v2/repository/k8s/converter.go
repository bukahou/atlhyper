package k8s

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// =============================================================================
// Pod 转换
// =============================================================================

// ConvertPod 转换 K8s Pod 到 model_v3（嵌套结构）
func ConvertPod(k8sPod *corev1.Pod) cluster.Pod {
	now := time.Now()
	createdAt := k8sPod.CreationTimestamp.Time
	age := formatDuration(now.Sub(createdAt))

	// 获取 Owner
	var ownerKind, ownerName string
	if len(k8sPod.OwnerReferences) > 0 {
		owner := k8sPod.OwnerReferences[0]
		ownerKind = owner.Kind
		ownerName = owner.Name
	}

	// 构建容器状态映射（用于合并 spec 和 status）
	containerStatusMap := make(map[string]corev1.ContainerStatus)
	for _, cs := range k8sPod.Status.ContainerStatuses {
		containerStatusMap[cs.Name] = cs
	}
	initContainerStatusMap := make(map[string]corev1.ContainerStatus)
	for _, cs := range k8sPod.Status.InitContainerStatuses {
		initContainerStatusMap[cs.Name] = cs
	}

	// 计算 ready 字符串和总重启次数
	totalContainers := len(k8sPod.Spec.Containers)
	readyContainers := 0
	var totalRestarts int32 = 0
	for _, cs := range k8sPod.Status.ContainerStatuses {
		if cs.Ready {
			readyContainers++
		}
		totalRestarts += cs.RestartCount
	}
	readyStr := ""
	if totalContainers > 0 {
		readyStr = strconv.Itoa(readyContainers) + "/" + strconv.Itoa(totalContainers)
	}

	// 获取 PodIPs
	var podIPs []string
	for _, ip := range k8sPod.Status.PodIPs {
		podIPs = append(podIPs, ip.IP)
	}

	// 获取状态原因和消息
	reason := ""
	message := ""
	for _, cond := range k8sPod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status != corev1.ConditionTrue {
			reason = cond.Reason
			message = cond.Message
			break
		}
	}
	// 如果 Pod 处于 Pending/Failed，尝试从 ContainerStatuses 获取原因
	// 优先取业务容器的异常原因，只有找不到时才回退到 sidecar
	if k8sPod.Status.Phase == corev1.PodPending || k8sPod.Status.Phase == corev1.PodFailed {
		var sidecarReason, sidecarMessage string
		for _, cs := range k8sPod.Status.ContainerStatuses {
			r, m := "", ""
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				r, m = cs.State.Waiting.Reason, cs.State.Waiting.Message
			} else if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
				r, m = cs.State.Terminated.Reason, cs.State.Terminated.Message
			}
			if r == "" {
				continue
			}
			if !model_v3.IsSidecarContainer(cs.Name) {
				reason, message = r, m
				break
			}
			if sidecarReason == "" {
				sidecarReason, sidecarMessage = r, m
			}
		}
		if reason == "" && sidecarReason != "" {
			reason, message = sidecarReason, sidecarMessage
		}
	}

	pod := cluster.Pod{
		// Summary
		Summary: cluster.PodSummary{
			Name:      k8sPod.Name,
			Namespace: k8sPod.Namespace,
			NodeName:  k8sPod.Spec.NodeName,
			OwnerKind: ownerKind,
			OwnerName: ownerName,
			CreatedAt: createdAt,
			Age:       age,
		},

		// Spec
		Spec: cluster.PodSpec{
			RestartPolicy:      string(k8sPod.Spec.RestartPolicy),
			ServiceAccountName: k8sPod.Spec.ServiceAccountName,
			NodeSelector:       k8sPod.Spec.NodeSelector,
			DNSPolicy:          string(k8sPod.Spec.DNSPolicy),
			HostNetwork:        k8sPod.Spec.HostNetwork,
		},

		// Status
		Status: cluster.PodStatus{
			Phase:    string(k8sPod.Status.Phase),
			Ready:    readyStr,
			Restarts: totalRestarts,
			QoSClass: string(k8sPod.Status.QOSClass),
			PodIP:    k8sPod.Status.PodIP,
			PodIPs:   podIPs,
			HostIP:   k8sPod.Status.HostIP,
			Reason:   reason,
			Message:  message,
		},

		// Labels & Annotations
		Labels:      k8sPod.Labels,
		Annotations: k8sPod.Annotations,
	}

	// Spec 补充字段
	if k8sPod.Spec.RuntimeClassName != nil {
		pod.Spec.RuntimeClassName = *k8sPod.Spec.RuntimeClassName
	}
	if k8sPod.Spec.PriorityClassName != "" {
		pod.Spec.PriorityClassName = k8sPod.Spec.PriorityClassName
	}
	if k8sPod.Spec.TerminationGracePeriodSeconds != nil {
		pod.Spec.TerminationGracePeriodSeconds = k8sPod.Spec.TerminationGracePeriodSeconds
	}

	// ImagePullSecrets
	for _, secret := range k8sPod.Spec.ImagePullSecrets {
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, secret.Name)
	}

	// Tolerations
	for _, t := range k8sPod.Spec.Tolerations {
		tol := cluster.Toleration{
			Key:      t.Key,
			Operator: string(t.Operator),
			Value:    t.Value,
			Effect:   string(t.Effect),
		}
		if t.TolerationSeconds != nil {
			tol.TolerationSeconds = t.TolerationSeconds
		}
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, tol)
	}

	// Affinity
	if k8sPod.Spec.Affinity != nil {
		pod.Spec.Affinity = &cluster.Affinity{}
		if k8sPod.Spec.Affinity.NodeAffinity != nil {
			pod.Spec.Affinity.NodeAffinity = "已配置"
		}
		if k8sPod.Spec.Affinity.PodAffinity != nil {
			pod.Spec.Affinity.PodAffinity = "已配置"
		}
		if k8sPod.Spec.Affinity.PodAntiAffinity != nil {
			pod.Spec.Affinity.PodAntiAffinity = "已配置"
		}
	}

	// Conditions
	for _, cond := range k8sPod.Status.Conditions {
		pod.Status.Conditions = append(pod.Status.Conditions, cluster.PodCondition{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastTransitionTime: cond.LastTransitionTime.Format(time.RFC3339),
		})
	}

	// Volumes
	for _, vol := range k8sPod.Spec.Volumes {
		v := cluster.VolumeSpec{Name: vol.Name}
		switch {
		case vol.ConfigMap != nil:
			v.Type = "ConfigMap"
			v.Source = vol.ConfigMap.Name
		case vol.Secret != nil:
			v.Type = "Secret"
			v.Source = vol.Secret.SecretName
		case vol.EmptyDir != nil:
			v.Type = "EmptyDir"
		case vol.PersistentVolumeClaim != nil:
			v.Type = "PVC"
			v.Source = vol.PersistentVolumeClaim.ClaimName
		case vol.HostPath != nil:
			v.Type = "HostPath"
			v.Source = vol.HostPath.Path
		case vol.Projected != nil:
			v.Type = "Projected"
		case vol.DownwardAPI != nil:
			v.Type = "DownwardAPI"
		default:
			v.Type = "Other"
		}
		pod.Volumes = append(pod.Volumes, v)
	}

	// Containers（合并 spec 和 status）
	for _, c := range k8sPod.Spec.Containers {
		pod.Containers = append(pod.Containers, convertPodContainer(&c, containerStatusMap[c.Name]))
	}

	// Init Containers
	for _, c := range k8sPod.Spec.InitContainers {
		pod.InitContainers = append(pod.InitContainers, convertPodContainer(&c, initContainerStatusMap[c.Name]))
	}

	return pod
}

// convertPodContainer 转换 Pod 容器（合并 spec 和 status）
func convertPodContainer(spec *corev1.Container, status corev1.ContainerStatus) cluster.PodContainerDetail {
	cd := cluster.PodContainerDetail{
		// 从 spec 获取
		Name:            spec.Name,
		Image:           spec.Image,
		ImagePullPolicy: string(spec.ImagePullPolicy),
		Command:         spec.Command,
		Args:            spec.Args,
		WorkingDir:      spec.WorkingDir,
	}

	// Ports
	for _, p := range spec.Ports {
		cd.Ports = append(cd.Ports, cluster.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      string(p.Protocol),
			HostPort:      p.HostPort,
		})
	}

	// Env
	for _, e := range spec.Env {
		ev := cluster.EnvVar{Name: e.Name, Value: e.Value}
		if e.ValueFrom != nil {
			if e.ValueFrom.ConfigMapKeyRef != nil {
				ev.ValueFrom = "configmap:" + e.ValueFrom.ConfigMapKeyRef.Name
			} else if e.ValueFrom.SecretKeyRef != nil {
				ev.ValueFrom = "secret:" + e.ValueFrom.SecretKeyRef.Name
			} else if e.ValueFrom.FieldRef != nil {
				ev.ValueFrom = "field:" + e.ValueFrom.FieldRef.FieldPath
			} else if e.ValueFrom.ResourceFieldRef != nil {
				ev.ValueFrom = "resource:" + e.ValueFrom.ResourceFieldRef.Resource
			}
		}
		cd.Envs = append(cd.Envs, ev)
	}

	// VolumeMounts
	for _, vm := range spec.VolumeMounts {
		cd.VolumeMounts = append(cd.VolumeMounts, cluster.VolumeMount{
			Name:      vm.Name,
			MountPath: vm.MountPath,
			SubPath:   vm.SubPath,
			ReadOnly:  vm.ReadOnly,
		})
	}

	// Resources
	if len(spec.Resources.Requests) > 0 {
		cd.Requests = make(map[string]string)
		for k, v := range spec.Resources.Requests {
			cd.Requests[string(k)] = v.String()
		}
	}
	if len(spec.Resources.Limits) > 0 {
		cd.Limits = make(map[string]string)
		for k, v := range spec.Resources.Limits {
			cd.Limits[string(k)] = v.String()
		}
	}

	// Probes
	cd.LivenessProbe = convertProbe(spec.LivenessProbe)
	cd.ReadinessProbe = convertProbe(spec.ReadinessProbe)
	cd.StartupProbe = convertProbe(spec.StartupProbe)

	// 从 status 获取运行状态
	if status.Name != "" {
		cd.Image = status.Image // 使用实际运行的镜像
		cd.Ready = status.Ready
		cd.RestartCount = status.RestartCount

		// 确定状态
		if status.State.Running != nil {
			cd.State = "running"
		} else if status.State.Waiting != nil {
			cd.State = "waiting"
			cd.StateReason = status.State.Waiting.Reason
			cd.StateMessage = status.State.Waiting.Message
		} else if status.State.Terminated != nil {
			cd.State = "terminated"
			cd.StateReason = status.State.Terminated.Reason
			cd.StateMessage = status.State.Terminated.Message
		}

		// 上次终止信息
		if status.LastTerminationState.Terminated != nil {
			cd.LastTerminationReason = status.LastTerminationState.Terminated.Reason
			cd.LastTerminationMessage = status.LastTerminationState.Terminated.Message
			if !status.LastTerminationState.Terminated.FinishedAt.IsZero() {
				cd.LastTerminationTime = status.LastTerminationState.Terminated.FinishedAt.Format(time.RFC3339)
			}
		}
	}

	return cd
}

// =============================================================================
// Node 转换
// =============================================================================

// ConvertNode 转换 K8s Node 到 model_v3
func ConvertNode(k8sNode *corev1.Node) cluster.Node {
	// 解析角色
	var roles []string
	for label := range k8sNode.Labels {
		if label == "node-role.kubernetes.io/master" {
			roles = append(roles, "master")
		} else if label == "node-role.kubernetes.io/control-plane" {
			roles = append(roles, "control-plane")
		} else if label == "node-role.kubernetes.io/worker" {
			roles = append(roles, "worker")
		}
	}

	// 计算 Ready 状态
	readyStatus := "Unknown"
	for _, cond := range k8sNode.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			readyStatus = string(cond.Status)
			break
		}
	}

	// 计算 Age
	age := time.Since(k8sNode.CreationTimestamp.Time)
	ageStr := formatDuration(age)

	node := cluster.Node{
		Summary: cluster.NodeSummary{
			Name:         k8sNode.Name,
			Roles:        roles,
			Ready:        readyStatus,
			Schedulable:  !k8sNode.Spec.Unschedulable,
			Age:          ageStr,
			CreationTime: k8sNode.CreationTimestamp.Time,
		},
		Spec: cluster.NodeSpec{
			PodCIDRs:      k8sNode.Spec.PodCIDRs,
			ProviderID:    k8sNode.Spec.ProviderID,
			Unschedulable: k8sNode.Spec.Unschedulable,
		},
		Capacity: cluster.NodeResources{
			CPU:              k8sNode.Status.Capacity.Cpu().String(),
			Memory:           k8sNode.Status.Capacity.Memory().String(),
			Pods:             k8sNode.Status.Capacity.Pods().String(),
			EphemeralStorage: k8sNode.Status.Capacity.StorageEphemeral().String(),
		},
		Allocatable: cluster.NodeResources{
			CPU:              k8sNode.Status.Allocatable.Cpu().String(),
			Memory:           k8sNode.Status.Allocatable.Memory().String(),
			Pods:             k8sNode.Status.Allocatable.Pods().String(),
			EphemeralStorage: k8sNode.Status.Allocatable.StorageEphemeral().String(),
		},
		Info: cluster.NodeInfo{
			OSImage:                 k8sNode.Status.NodeInfo.OSImage,
			OperatingSystem:         k8sNode.Status.NodeInfo.OperatingSystem,
			Architecture:            k8sNode.Status.NodeInfo.Architecture,
			KernelVersion:           k8sNode.Status.NodeInfo.KernelVersion,
			ContainerRuntimeVersion: k8sNode.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:          k8sNode.Status.NodeInfo.KubeletVersion,
			KubeProxyVersion:        k8sNode.Status.NodeInfo.KubeProxyVersion,
		},
		Labels: k8sNode.Labels,
	}

	// 获取地址（优先使用 IPv4 地址）
	for _, addr := range k8sNode.Status.Addresses {
		switch addr.Type {
		case corev1.NodeHostName:
			node.Addresses.Hostname = addr.Address
		case corev1.NodeInternalIP:
			// 优先保留 IPv4 地址（不含冒号），只有当前为空或当前是 IPv6 时才覆盖
			if node.Addresses.InternalIP == "" || (strings.Contains(node.Addresses.InternalIP, ":") && !strings.Contains(addr.Address, ":")) {
				node.Addresses.InternalIP = addr.Address
			}
		case corev1.NodeExternalIP:
			// 同样优先保留 IPv4 地址
			if node.Addresses.ExternalIP == "" || (strings.Contains(node.Addresses.ExternalIP, ":") && !strings.Contains(addr.Address, ":")) {
				node.Addresses.ExternalIP = addr.Address
			}
		}
		node.Addresses.All = append(node.Addresses.All, cluster.NodeAddr{
			Type:    string(addr.Type),
			Address: addr.Address,
		})
	}

	// 转换 Conditions
	for _, cond := range k8sNode.Status.Conditions {
		node.Conditions = append(node.Conditions, cluster.NodeCondition{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastHeartbeatTime:  cond.LastHeartbeatTime.Time,
			LastTransitionTime: cond.LastTransitionTime.Time,
		})
	}

	// 转换 Taints
	for _, taint := range k8sNode.Spec.Taints {
		nt := cluster.NodeTaint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		}
		if taint.TimeAdded != nil {
			t := taint.TimeAdded.Time
			nt.TimeAdded = &t
		}
		node.Taints = append(node.Taints, nt)
	}

	return node
}

// formatDuration 格式化时间间隔为人类可读格式
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "0m"
}

// =============================================================================
// Deployment 转换
// =============================================================================

// ConvertDeployment 转换 K8s Deployment 到 model_v3
func ConvertDeployment(k8sDeploy *appsv1.Deployment) cluster.Deployment {
	age := time.Since(k8sDeploy.CreationTimestamp.Time)
	ageStr := formatDuration(age)

	// 构建 selector 字符串
	selectorStr := ""
	if k8sDeploy.Spec.Selector != nil {
		parts := make([]string, 0, len(k8sDeploy.Spec.Selector.MatchLabels))
		for k, v := range k8sDeploy.Spec.Selector.MatchLabels {
			parts = append(parts, k+"="+v)
		}
		selectorStr = strings.Join(parts, ",")
	}

	replicas := int32(0)
	if k8sDeploy.Spec.Replicas != nil {
		replicas = *k8sDeploy.Spec.Replicas
	}

	deploy := cluster.Deployment{
		Summary: cluster.DeploymentSummary{
			Name:        k8sDeploy.Name,
			Namespace:   k8sDeploy.Namespace,
			Strategy:    string(k8sDeploy.Spec.Strategy.Type),
			Replicas:    replicas,
			Updated:     k8sDeploy.Status.UpdatedReplicas,
			Ready:       k8sDeploy.Status.ReadyReplicas,
			Available:   k8sDeploy.Status.AvailableReplicas,
			Unavailable: k8sDeploy.Status.UnavailableReplicas,
			Paused:      k8sDeploy.Spec.Paused,
			CreatedAt:   k8sDeploy.CreationTimestamp.Time,
			Age:         ageStr,
			Selector:    selectorStr,
		},
		Labels:      k8sDeploy.Labels,
		Annotations: k8sDeploy.Annotations,
	}

	// Spec
	deploy.Spec = cluster.DeploymentSpec{
		Replicas:                k8sDeploy.Spec.Replicas,
		MinReadySeconds:         k8sDeploy.Spec.MinReadySeconds,
		RevisionHistoryLimit:    k8sDeploy.Spec.RevisionHistoryLimit,
		ProgressDeadlineSeconds: k8sDeploy.Spec.ProgressDeadlineSeconds,
	}

	// Selector
	if k8sDeploy.Spec.Selector != nil {
		deploy.Spec.Selector = &cluster.LabelSelector{
			MatchLabels: k8sDeploy.Spec.Selector.MatchLabels,
		}
		for _, expr := range k8sDeploy.Spec.Selector.MatchExpressions {
			deploy.Spec.Selector.MatchExpressions = append(deploy.Spec.Selector.MatchExpressions, cluster.LabelExpr{
				Key:      expr.Key,
				Operator: string(expr.Operator),
				Values:   expr.Values,
			})
		}
	}

	// Strategy
	deploy.Spec.Strategy = &cluster.DeploymentStrategy{
		Type: string(k8sDeploy.Spec.Strategy.Type),
	}
	if k8sDeploy.Spec.Strategy.RollingUpdate != nil {
		deploy.Spec.Strategy.RollingUpdate = &cluster.RollingUpdateStrategy{}
		if k8sDeploy.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			deploy.Spec.Strategy.RollingUpdate.MaxUnavailable = k8sDeploy.Spec.Strategy.RollingUpdate.MaxUnavailable.String()
		}
		if k8sDeploy.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			deploy.Spec.Strategy.RollingUpdate.MaxSurge = k8sDeploy.Spec.Strategy.RollingUpdate.MaxSurge.String()
		}
	}

	// Status
	deploy.Status = cluster.DeploymentStatus{
		ObservedGeneration:  k8sDeploy.Status.ObservedGeneration,
		Replicas:            k8sDeploy.Status.Replicas,
		UpdatedReplicas:     k8sDeploy.Status.UpdatedReplicas,
		ReadyReplicas:       k8sDeploy.Status.ReadyReplicas,
		AvailableReplicas:   k8sDeploy.Status.AvailableReplicas,
		UnavailableReplicas: k8sDeploy.Status.UnavailableReplicas,
		CollisionCount:      k8sDeploy.Status.CollisionCount,
	}

	// Conditions
	for _, cond := range k8sDeploy.Status.Conditions {
		deploy.Status.Conditions = append(deploy.Status.Conditions, cluster.DeploymentCondition{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastUpdateTime:     cond.LastUpdateTime.Time,
			LastTransitionTime: cond.LastTransitionTime.Time,
		})
	}

	// Template
	deploy.Template = convertPodTemplate(&k8sDeploy.Spec.Template.Spec)
	deploy.Template.Labels = k8sDeploy.Spec.Template.Labels
	deploy.Template.Annotations = k8sDeploy.Spec.Template.Annotations

	// Rollout phase
	deploy.Rollout = determineRolloutPhase(k8sDeploy)

	return deploy
}

// convertPodTemplate 转换 Pod 模板
func convertPodTemplate(spec *corev1.PodSpec) cluster.PodTemplate {
	tpl := cluster.PodTemplate{
		ServiceAccountName: spec.ServiceAccountName,
		NodeSelector:       spec.NodeSelector,
		RuntimeClassName:   "",
		HostNetwork:        spec.HostNetwork,
		DNSPolicy:          string(spec.DNSPolicy),
	}

	if spec.RuntimeClassName != nil {
		tpl.RuntimeClassName = *spec.RuntimeClassName
	}

	// ImagePullSecrets
	for _, secret := range spec.ImagePullSecrets {
		tpl.ImagePullSecrets = append(tpl.ImagePullSecrets, secret.Name)
	}

	// Tolerations
	for _, t := range spec.Tolerations {
		tol := cluster.Toleration{
			Key:      t.Key,
			Operator: string(t.Operator),
			Value:    t.Value,
			Effect:   string(t.Effect),
		}
		if t.TolerationSeconds != nil {
			tol.TolerationSeconds = t.TolerationSeconds
		}
		tpl.Tolerations = append(tpl.Tolerations, tol)
	}

	// Affinity (简化描述)
	if spec.Affinity != nil {
		tpl.Affinity = &cluster.Affinity{}
		if spec.Affinity.NodeAffinity != nil {
			tpl.Affinity.NodeAffinity = "已配置"
		}
		if spec.Affinity.PodAffinity != nil {
			tpl.Affinity.PodAffinity = "已配置"
		}
		if spec.Affinity.PodAntiAffinity != nil {
			tpl.Affinity.PodAntiAffinity = "已配置"
		}
	}

	// Volumes
	for _, vol := range spec.Volumes {
		v := cluster.VolumeSpec{Name: vol.Name}
		switch {
		case vol.ConfigMap != nil:
			v.Type = "ConfigMap"
			v.Source = vol.ConfigMap.Name
		case vol.Secret != nil:
			v.Type = "Secret"
			v.Source = vol.Secret.SecretName
		case vol.EmptyDir != nil:
			v.Type = "EmptyDir"
		case vol.PersistentVolumeClaim != nil:
			v.Type = "PVC"
			v.Source = vol.PersistentVolumeClaim.ClaimName
		case vol.HostPath != nil:
			v.Type = "HostPath"
			v.Source = vol.HostPath.Path
		case vol.Projected != nil:
			v.Type = "Projected"
		case vol.DownwardAPI != nil:
			v.Type = "DownwardAPI"
		default:
			v.Type = "Other"
		}
		tpl.Volumes = append(tpl.Volumes, v)
	}

	// Containers
	for _, c := range spec.Containers {
		tpl.Containers = append(tpl.Containers, convertContainerDetail(&c))
	}

	return tpl
}

// convertContainerDetail 转换容器详情
func convertContainerDetail(c *corev1.Container) cluster.ContainerDetail {
	cd := cluster.ContainerDetail{
		Name:            c.Name,
		Image:           c.Image,
		ImagePullPolicy: string(c.ImagePullPolicy),
		Command:         c.Command,
		Args:            c.Args,
		WorkingDir:      c.WorkingDir,
	}

	// Ports
	for _, p := range c.Ports {
		cd.Ports = append(cd.Ports, cluster.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      string(p.Protocol),
			HostPort:      p.HostPort,
		})
	}

	// Env
	for _, e := range c.Env {
		ev := cluster.EnvVar{Name: e.Name, Value: e.Value}
		if e.ValueFrom != nil {
			if e.ValueFrom.ConfigMapKeyRef != nil {
				ev.ValueFrom = "configmap:" + e.ValueFrom.ConfigMapKeyRef.Name
			} else if e.ValueFrom.SecretKeyRef != nil {
				ev.ValueFrom = "secret:" + e.ValueFrom.SecretKeyRef.Name
			} else if e.ValueFrom.FieldRef != nil {
				ev.ValueFrom = "field:" + e.ValueFrom.FieldRef.FieldPath
			} else if e.ValueFrom.ResourceFieldRef != nil {
				ev.ValueFrom = "resource:" + e.ValueFrom.ResourceFieldRef.Resource
			}
		}
		cd.Envs = append(cd.Envs, ev)
	}

	// VolumeMounts
	for _, vm := range c.VolumeMounts {
		cd.VolumeMounts = append(cd.VolumeMounts, cluster.VolumeMount{
			Name:      vm.Name,
			MountPath: vm.MountPath,
			SubPath:   vm.SubPath,
			ReadOnly:  vm.ReadOnly,
		})
	}

	// Resources
	if len(c.Resources.Requests) > 0 {
		cd.Requests = make(map[string]string)
		for k, v := range c.Resources.Requests {
			cd.Requests[string(k)] = v.String()
		}
	}
	if len(c.Resources.Limits) > 0 {
		cd.Limits = make(map[string]string)
		for k, v := range c.Resources.Limits {
			cd.Limits[string(k)] = v.String()
		}
	}

	// Probes
	cd.LivenessProbe = convertProbe(c.LivenessProbe)
	cd.ReadinessProbe = convertProbe(c.ReadinessProbe)
	cd.StartupProbe = convertProbe(c.StartupProbe)

	return cd
}

// convertProbe 转换探针
func convertProbe(probe *corev1.Probe) *cluster.Probe {
	if probe == nil {
		return nil
	}
	p := &cluster.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}
	if probe.HTTPGet != nil {
		p.Type = "httpGet"
		p.Path = probe.HTTPGet.Path
		p.Port = probe.HTTPGet.Port.IntVal
	} else if probe.TCPSocket != nil {
		p.Type = "tcpSocket"
		p.Port = probe.TCPSocket.Port.IntVal
	} else if probe.Exec != nil {
		p.Type = "exec"
		p.Command = strings.Join(probe.Exec.Command, " ")
	}
	return p
}

// determineRolloutPhase 判断 Rollout 阶段
func determineRolloutPhase(deploy *appsv1.Deployment) *cluster.DeploymentRollout {
	rollout := &cluster.DeploymentRollout{
		Phase: "Unknown",
	}

	for _, cond := range deploy.Status.Conditions {
		switch cond.Type {
		case appsv1.DeploymentAvailable:
			if cond.Status == corev1.ConditionTrue {
				rollout.Phase = "Available"
				rollout.Badges = append(rollout.Badges, "Available")
			}
		case appsv1.DeploymentProgressing:
			if cond.Status == corev1.ConditionTrue {
				if rollout.Phase != "Available" {
					rollout.Phase = "Progressing"
				}
				rollout.Badges = append(rollout.Badges, "Progressing")
				rollout.Message = cond.Message
			} else if cond.Reason == "ProgressDeadlineExceeded" {
				rollout.Phase = "Failed"
				rollout.Badges = append(rollout.Badges, "Failed")
				rollout.Message = cond.Message
			}
		case appsv1.DeploymentReplicaFailure:
			if cond.Status == corev1.ConditionTrue {
				rollout.Phase = "Failed"
				rollout.Badges = append(rollout.Badges, "ReplicaFailure")
				rollout.Message = cond.Message
			}
		}
	}

	if deploy.Spec.Paused {
		rollout.Badges = append(rollout.Badges, "Paused")
	}

	return rollout
}

// =============================================================================
// Service 转换
// =============================================================================

// ConvertService 转换 K8s Service 到 model_v3（嵌套结构）
func ConvertService(k8sSvc *corev1.Service) cluster.Service {
	now := time.Now()
	createdAt := k8sSvc.CreationTimestamp.Time
	age := formatDuration(now.Sub(createdAt))

	svcType := string(k8sSvc.Spec.Type)
	if svcType == "" {
		svcType = "ClusterIP"
	}

	// 构建 badges
	badges := make([]string, 0)
	if svcType == "LoadBalancer" {
		badges = append(badges, "LB")
	} else if svcType == "NodePort" {
		badges = append(badges, "NodePort")
	} else if k8sSvc.Spec.ClusterIP == "None" {
		badges = append(badges, "Headless")
	}
	if k8sSvc.Spec.ExternalName != "" {
		badges = append(badges, "ExternalName")
	}

	svc := cluster.Service{
		// Summary
		Summary: cluster.ServiceSummary{
			Name:         k8sSvc.Name,
			Namespace:    k8sSvc.Namespace,
			Type:         svcType,
			CreatedAt:    createdAt,
			Age:          age,
			PortsCount:   len(k8sSvc.Spec.Ports),
			HasSelector:  len(k8sSvc.Spec.Selector) > 0,
			Badges:       badges,
			ClusterIP:    k8sSvc.Spec.ClusterIP,
			ExternalName: k8sSvc.Spec.ExternalName,
		},

		// Spec
		Spec: convertServiceSpec(k8sSvc),

		// Selector
		Selector: k8sSvc.Spec.Selector,

		// Network
		Network: convertServiceNetwork(k8sSvc),

		// 元数据
		Labels:      k8sSvc.Labels,
		Annotations: k8sSvc.Annotations,
	}

	// 端口
	for _, p := range k8sSvc.Spec.Ports {
		appProtocol := ""
		if p.AppProtocol != nil {
			appProtocol = *p.AppProtocol
		}
		svc.Ports = append(svc.Ports, cluster.ServicePort{
			Name:        p.Name,
			Protocol:    string(p.Protocol),
			Port:        p.Port,
			TargetPort:  p.TargetPort.String(),
			NodePort:    p.NodePort,
			AppProtocol: appProtocol,
		})
	}

	return svc
}

// convertServiceSpec 转换 Service Spec
func convertServiceSpec(k8sSvc *corev1.Service) cluster.ServiceSpec {
	spec := cluster.ServiceSpec{
		Type:            string(k8sSvc.Spec.Type),
		SessionAffinity: string(k8sSvc.Spec.SessionAffinity),
		ExternalName:    k8sSvc.Spec.ExternalName,
	}

	// Session Affinity Timeout
	if k8sSvc.Spec.SessionAffinityConfig != nil &&
		k8sSvc.Spec.SessionAffinityConfig.ClientIP != nil &&
		k8sSvc.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds != nil {
		timeout := *k8sSvc.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds
		spec.SessionAffinityTimeoutSeconds = &timeout
	}

	// Traffic Policies
	if k8sSvc.Spec.ExternalTrafficPolicy != "" {
		spec.ExternalTrafficPolicy = string(k8sSvc.Spec.ExternalTrafficPolicy)
	}
	if k8sSvc.Spec.InternalTrafficPolicy != nil {
		spec.InternalTrafficPolicy = string(*k8sSvc.Spec.InternalTrafficPolicy)
	}

	// IP Families
	for _, f := range k8sSvc.Spec.IPFamilies {
		spec.IPFamilies = append(spec.IPFamilies, string(f))
	}
	if k8sSvc.Spec.IPFamilyPolicy != nil {
		spec.IPFamilyPolicy = string(*k8sSvc.Spec.IPFamilyPolicy)
	}

	// IPs
	spec.ClusterIPs = k8sSvc.Spec.ClusterIPs
	spec.ExternalIPs = k8sSvc.Spec.ExternalIPs

	// LoadBalancer 相关
	if k8sSvc.Spec.LoadBalancerClass != nil {
		spec.LoadBalancerClass = *k8sSvc.Spec.LoadBalancerClass
	}
	spec.LoadBalancerSourceRanges = k8sSvc.Spec.LoadBalancerSourceRanges
	spec.PublishNotReadyAddresses = k8sSvc.Spec.PublishNotReadyAddresses
	if k8sSvc.Spec.AllocateLoadBalancerNodePorts != nil {
		spec.AllocateLoadBalancerNodePorts = k8sSvc.Spec.AllocateLoadBalancerNodePorts
	}
	spec.HealthCheckNodePort = k8sSvc.Spec.HealthCheckNodePort

	return spec
}

// convertServiceNetwork 转换 Service Network 信息
func convertServiceNetwork(k8sSvc *corev1.Service) cluster.ServiceNetwork {
	network := cluster.ServiceNetwork{
		ClusterIPs:  k8sSvc.Spec.ClusterIPs,
		ExternalIPs: k8sSvc.Spec.ExternalIPs,
	}

	// LoadBalancer Ingress
	for _, ing := range k8sSvc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			network.LoadBalancerIngress = append(network.LoadBalancerIngress, ing.IP)
		} else if ing.Hostname != "" {
			network.LoadBalancerIngress = append(network.LoadBalancerIngress, ing.Hostname)
		}
	}

	// IP Families
	for _, f := range k8sSvc.Spec.IPFamilies {
		network.IPFamilies = append(network.IPFamilies, string(f))
	}
	if k8sSvc.Spec.IPFamilyPolicy != nil {
		network.IPFamilyPolicy = string(*k8sSvc.Spec.IPFamilyPolicy)
	}

	// Traffic Policies
	if k8sSvc.Spec.ExternalTrafficPolicy != "" {
		network.ExternalTrafficPolicy = string(k8sSvc.Spec.ExternalTrafficPolicy)
	}
	if k8sSvc.Spec.InternalTrafficPolicy != nil {
		network.InternalTrafficPolicy = string(*k8sSvc.Spec.InternalTrafficPolicy)
	}

	return network
}

// =============================================================================
// Event 转换
// =============================================================================

// ConvertEvent 转换 K8s Event 到 model_v3
func ConvertEvent(k8sEvent *corev1.Event) cluster.Event {
	event := cluster.Event{
		CommonMeta: buildCommonMeta(
			string(k8sEvent.UID),
			k8sEvent.Name,
			k8sEvent.Namespace,
			"Event",
			nil,
			k8sEvent.CreationTimestamp.Time,
		),
		Type:    k8sEvent.Type,
		Reason:  k8sEvent.Reason,
		Message: k8sEvent.Message,
		Count:   k8sEvent.Count,
		Source:  k8sEvent.Source.Component,
		InvolvedObject: model_v3.ResourceRef{
			Kind:      k8sEvent.InvolvedObject.Kind,
			Name:      k8sEvent.InvolvedObject.Name,
			Namespace: k8sEvent.InvolvedObject.Namespace,
			UID:       string(k8sEvent.InvolvedObject.UID),
		},
	}

	// 设置时间戳
	if !k8sEvent.FirstTimestamp.IsZero() {
		event.FirstTimestamp = k8sEvent.FirstTimestamp.Time
	}
	if !k8sEvent.LastTimestamp.IsZero() {
		event.LastTimestamp = k8sEvent.LastTimestamp.Time
	}

	return event
}

// =============================================================================
// Namespace 转换
// =============================================================================

// ConvertNamespace 转换 K8s Namespace 到 model_v3
//
// 转换基本信息和状态，Resources 字段将在
// snapshotService.calculateNamespaceResources 中填充。
func ConvertNamespace(k8sNs *corev1.Namespace) cluster.Namespace {
	now := time.Now()
	createdAt := k8sNs.CreationTimestamp.Time
	age := formatDuration(now.Sub(createdAt))

	ns := cluster.Namespace{
		Summary: cluster.NamespaceSummary{
			Name:      k8sNs.Name,
			CreatedAt: k8sNs.CreationTimestamp.Format(time.RFC3339),
			Age:       age,
		},
		Status: cluster.NamespaceStatus{
			Phase: string(k8sNs.Status.Phase),
		},
		Labels:      k8sNs.Labels,
		Annotations: k8sNs.Annotations,
	}

	return ns
}

// =============================================================================
// ConfigMap 转换
// =============================================================================

// ConvertConfigMap 转换 K8s ConfigMap 到 model_v3
func ConvertConfigMap(k8sCm *corev1.ConfigMap) cluster.ConfigMap {
	cm := cluster.ConfigMap{
		CommonMeta: buildCommonMeta(
			string(k8sCm.UID),
			k8sCm.Name,
			k8sCm.Namespace,
			"ConfigMap",
			k8sCm.Labels,
			k8sCm.CreationTimestamp.Time,
		),
	}

	// 只存 key，不存 value
	for key := range k8sCm.Data {
		cm.DataKeys = append(cm.DataKeys, key)
	}

	return cm
}

// =============================================================================
// Secret 转换
// =============================================================================

// ConvertSecret 转换 K8s Secret 到 model_v3
func ConvertSecret(k8sSecret *corev1.Secret) cluster.Secret {
	secret := cluster.Secret{
		CommonMeta: buildCommonMeta(
			string(k8sSecret.UID),
			k8sSecret.Name,
			k8sSecret.Namespace,
			"Secret",
			k8sSecret.Labels,
			k8sSecret.CreationTimestamp.Time,
		),
		Type: string(k8sSecret.Type),
	}

	// 只存 key，不存 value
	for key := range k8sSecret.Data {
		secret.DataKeys = append(secret.DataKeys, key)
	}

	return secret
}

// =============================================================================
// Ingress 转换
// =============================================================================

// ConvertIngress 转换 K8s Ingress 到 model_v3（嵌套结构）
func ConvertIngress(k8sIng *networkingv1.Ingress) cluster.Ingress {
	now := time.Now()
	createdAt := k8sIng.CreationTimestamp.Time
	age := formatDuration(now.Sub(createdAt))

	// 收集 hosts 和统计 paths
	var hosts []string
	pathsCount := 0
	for _, rule := range k8sIng.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
		if rule.HTTP != nil {
			pathsCount += len(rule.HTTP.Paths)
		}
	}

	// 获取 IngressClass（优先 spec.ingressClassName，其次 annotation）
	ingressClass := ""
	if k8sIng.Spec.IngressClassName != nil {
		ingressClass = *k8sIng.Spec.IngressClassName
	} else if v, ok := k8sIng.Annotations["kubernetes.io/ingress.class"]; ok {
		ingressClass = v
	}

	// 构建 Ingress
	ing := cluster.Ingress{
		Summary: cluster.IngressSummary{
			Name:         k8sIng.Name,
			Namespace:    k8sIng.Namespace,
			CreatedAt:    createdAt,
			Age:          age,
			IngressClass: ingressClass,
			HostsCount:   len(hosts),
			PathsCount:   pathsCount,
			TLSEnabled:   len(k8sIng.Spec.TLS) > 0,
			Hosts:        hosts,
		},
		Spec: cluster.IngressSpec{
			IngressClassName: ingressClass,
		},
		Labels:      k8sIng.Labels,
		Annotations: k8sIng.Annotations,
	}

	// 默认后端
	if k8sIng.Spec.DefaultBackend != nil {
		ing.Spec.DefaultBackend = convertIngressBackend(k8sIng.Spec.DefaultBackend)
	}

	// 规则
	for _, rule := range k8sIng.Spec.Rules {
		r := cluster.IngressRule{
			Host: rule.Host,
		}
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				p := cluster.IngressPath{
					Path:     path.Path,
					PathType: string(*path.PathType),
					Backend:  convertIngressBackend(&path.Backend),
				}
				r.Paths = append(r.Paths, p)
			}
		}
		ing.Spec.Rules = append(ing.Spec.Rules, r)
	}

	// TLS
	for _, tls := range k8sIng.Spec.TLS {
		ing.Spec.TLS = append(ing.Spec.TLS, cluster.IngressTLS{
			Hosts:      tls.Hosts,
			SecretName: tls.SecretName,
		})
	}

	// Status - LoadBalancer IPs
	for _, ingress := range k8sIng.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			ing.Status.LoadBalancer = append(ing.Status.LoadBalancer, ingress.IP)
		} else if ingress.Hostname != "" {
			ing.Status.LoadBalancer = append(ing.Status.LoadBalancer, ingress.Hostname)
		}
	}

	return ing
}

// convertIngressBackend 转换 Ingress Backend
func convertIngressBackend(backend *networkingv1.IngressBackend) *cluster.IngressBackend {
	if backend == nil {
		return nil
	}

	b := &cluster.IngressBackend{}

	if backend.Service != nil {
		b.Type = "Service"
		b.Service = &cluster.IngressServiceBackend{
			Name: backend.Service.Name,
		}
		if backend.Service.Port.Name != "" {
			b.Service.PortName = backend.Service.Port.Name
		}
		if backend.Service.Port.Number != 0 {
			b.Service.PortNumber = backend.Service.Port.Number
		}
	}

	if backend.Resource != nil {
		b.Type = "Resource"
		b.Resource = &cluster.IngressResourceRef{
			Kind: backend.Resource.Kind,
			Name: backend.Resource.Name,
		}
		if backend.Resource.APIGroup != nil {
			b.Resource.APIGroup = *backend.Resource.APIGroup
		}
	}

	return b
}

// =============================================================================
// StatefulSet 转换
// =============================================================================

// ConvertStatefulSet 转换 K8s StatefulSet 到 model_v3（嵌套结构）
func ConvertStatefulSet(k8sSts *appsv1.StatefulSet) cluster.StatefulSet {
	now := time.Now()
	createdAt := k8sSts.CreationTimestamp.Time
	age := formatDuration(now.Sub(createdAt))

	replicas := int32(1)
	if k8sSts.Spec.Replicas != nil {
		replicas = *k8sSts.Spec.Replicas
	}

	// 构建 selector 字符串
	selectorStr := ""
	if k8sSts.Spec.Selector != nil && len(k8sSts.Spec.Selector.MatchLabels) > 0 {
		pairs := make([]string, 0)
		for k, v := range k8sSts.Spec.Selector.MatchLabels {
			pairs = append(pairs, k+"="+v)
		}
		selectorStr = strings.Join(pairs, ",")
	}

	sts := cluster.StatefulSet{
		// Summary
		Summary: cluster.StatefulSetSummary{
			Name:        k8sSts.Name,
			Namespace:   k8sSts.Namespace,
			Replicas:    replicas,
			Ready:       k8sSts.Status.ReadyReplicas,
			Current:     k8sSts.Status.CurrentReplicas,
			Updated:     k8sSts.Status.UpdatedReplicas,
			Available:   k8sSts.Status.AvailableReplicas,
			CreatedAt:   createdAt,
			Age:         age,
			ServiceName: k8sSts.Spec.ServiceName,
			Selector:    selectorStr,
		},

		// Spec
		Spec: convertStatefulSetSpec(k8sSts),

		// Template
		Template: convertPodTemplate(&k8sSts.Spec.Template.Spec),

		// Status
		Status: convertStatefulSetStatus(k8sSts),

		// Rollout
		Rollout: determineStatefulSetRollout(k8sSts),

		// 元数据
		Labels:      k8sSts.Labels,
		Annotations: k8sSts.Annotations,
	}

	return sts
}

// convertStatefulSetSpec 转换 StatefulSet Spec
func convertStatefulSetSpec(k8sSts *appsv1.StatefulSet) cluster.StatefulSetSpec {
	spec := cluster.StatefulSetSpec{
		Replicas:             k8sSts.Spec.Replicas,
		ServiceName:          k8sSts.Spec.ServiceName,
		PodManagementPolicy:  string(k8sSts.Spec.PodManagementPolicy),
		MinReadySeconds:      k8sSts.Spec.MinReadySeconds,
		RevisionHistoryLimit: k8sSts.Spec.RevisionHistoryLimit,
	}

	// Update Strategy
	if k8sSts.Spec.UpdateStrategy.Type != "" {
		spec.UpdateStrategy = &cluster.UpdateStrategy{
			Type: string(k8sSts.Spec.UpdateStrategy.Type),
		}
		if k8sSts.Spec.UpdateStrategy.RollingUpdate != nil {
			spec.UpdateStrategy.Partition = k8sSts.Spec.UpdateStrategy.RollingUpdate.Partition
			if k8sSts.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable != nil {
				spec.UpdateStrategy.MaxUnavailable = k8sSts.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.String()
			}
		}
	}

	// Selector
	if k8sSts.Spec.Selector != nil {
		spec.Selector = &cluster.LabelSelector{
			MatchLabels: k8sSts.Spec.Selector.MatchLabels,
		}
	}

	// PVC Retention Policy
	if k8sSts.Spec.PersistentVolumeClaimRetentionPolicy != nil {
		spec.PersistentVolumeClaimRetentionPolicy = &cluster.PVCRetentionPolicy{
			WhenDeleted: string(k8sSts.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted),
			WhenScaled:  string(k8sSts.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled),
		}
	}

	// Volume Claim Templates
	for _, pvc := range k8sSts.Spec.VolumeClaimTemplates {
		vct := cluster.VolumeClaimTemplate{
			Name: pvc.Name,
		}
		for _, am := range pvc.Spec.AccessModes {
			vct.AccessModes = append(vct.AccessModes, string(am))
		}
		if pvc.Spec.StorageClassName != nil {
			vct.StorageClass = *pvc.Spec.StorageClassName
		}
		if pvc.Spec.Resources.Requests != nil {
			if storage, ok := pvc.Spec.Resources.Requests["storage"]; ok {
				vct.Storage = storage.String()
			}
		}
		spec.VolumeClaimTemplates = append(spec.VolumeClaimTemplates, vct)
	}

	return spec
}

// convertStatefulSetStatus 转换 StatefulSet Status
func convertStatefulSetStatus(k8sSts *appsv1.StatefulSet) cluster.StatefulSetStatus {
	status := cluster.StatefulSetStatus{
		ObservedGeneration: k8sSts.Status.ObservedGeneration,
		Replicas:           k8sSts.Status.Replicas,
		ReadyReplicas:      k8sSts.Status.ReadyReplicas,
		CurrentReplicas:    k8sSts.Status.CurrentReplicas,
		UpdatedReplicas:    k8sSts.Status.UpdatedReplicas,
		AvailableReplicas:  k8sSts.Status.AvailableReplicas,
		CurrentRevision:    k8sSts.Status.CurrentRevision,
		UpdateRevision:     k8sSts.Status.UpdateRevision,
		CollisionCount:     k8sSts.Status.CollisionCount,
	}

	// Conditions
	for _, c := range k8sSts.Status.Conditions {
		status.Conditions = append(status.Conditions, cluster.WorkloadCondition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
			LastTransitionTime: c.LastTransitionTime.Format(time.RFC3339),
		})
	}

	return status
}

// determineStatefulSetRollout 判断 StatefulSet 发布状态
func determineStatefulSetRollout(k8sSts *appsv1.StatefulSet) *cluster.WorkloadRollout {
	rollout := &cluster.WorkloadRollout{
		Phase:  "Complete",
		Badges: make([]string, 0),
	}

	replicas := int32(1)
	if k8sSts.Spec.Replicas != nil {
		replicas = *k8sSts.Spec.Replicas
	}

	// 判断状态
	if k8sSts.Status.UpdatedReplicas < replicas {
		rollout.Phase = "Progressing"
		rollout.Message = "Rolling update in progress"
		rollout.Badges = append(rollout.Badges, "Updating")
	} else if k8sSts.Status.ReadyReplicas < replicas {
		rollout.Phase = "Progressing"
		rollout.Message = "Waiting for pods to be ready"
		rollout.Badges = append(rollout.Badges, "Scaling")
	} else if k8sSts.Status.ReadyReplicas == 0 && replicas > 0 {
		rollout.Phase = "Degraded"
		rollout.Message = "No pods are ready"
	}

	if k8sSts.Status.CurrentRevision != k8sSts.Status.UpdateRevision {
		rollout.Badges = append(rollout.Badges, "NewRevision")
	}

	return rollout
}

// =============================================================================
// DaemonSet 转换
// =============================================================================

// ConvertDaemonSet 转换 K8s DaemonSet 到 model_v3（嵌套结构）
func ConvertDaemonSet(k8sDs *appsv1.DaemonSet) cluster.DaemonSet {
	now := time.Now()
	createdAt := k8sDs.CreationTimestamp.Time
	age := formatDuration(now.Sub(createdAt))

	// 构建 selector 字符串
	selectorStr := ""
	if k8sDs.Spec.Selector != nil && len(k8sDs.Spec.Selector.MatchLabels) > 0 {
		pairs := make([]string, 0)
		for k, v := range k8sDs.Spec.Selector.MatchLabels {
			pairs = append(pairs, k+"="+v)
		}
		selectorStr = strings.Join(pairs, ",")
	}

	ds := cluster.DaemonSet{
		// Summary
		Summary: cluster.DaemonSetSummary{
			Name:                   k8sDs.Name,
			Namespace:              k8sDs.Namespace,
			DesiredNumberScheduled: k8sDs.Status.DesiredNumberScheduled,
			CurrentNumberScheduled: k8sDs.Status.CurrentNumberScheduled,
			NumberReady:            k8sDs.Status.NumberReady,
			NumberAvailable:        k8sDs.Status.NumberAvailable,
			NumberUnavailable:      k8sDs.Status.NumberUnavailable,
			NumberMisscheduled:     k8sDs.Status.NumberMisscheduled,
			UpdatedNumberScheduled: k8sDs.Status.UpdatedNumberScheduled,
			CreatedAt:              createdAt,
			Age:                    age,
			Selector:               selectorStr,
		},

		// Spec
		Spec: convertDaemonSetSpec(k8sDs),

		// Template
		Template: convertPodTemplate(&k8sDs.Spec.Template.Spec),

		// Status
		Status: convertDaemonSetStatus(k8sDs),

		// Rollout
		Rollout: determineDaemonSetRollout(k8sDs),

		// 元数据
		Labels:      k8sDs.Labels,
		Annotations: k8sDs.Annotations,
	}

	return ds
}

// convertDaemonSetSpec 转换 DaemonSet Spec
func convertDaemonSetSpec(k8sDs *appsv1.DaemonSet) cluster.DaemonSetSpec {
	spec := cluster.DaemonSetSpec{
		MinReadySeconds:      k8sDs.Spec.MinReadySeconds,
		RevisionHistoryLimit: k8sDs.Spec.RevisionHistoryLimit,
	}

	// Update Strategy
	if k8sDs.Spec.UpdateStrategy.Type != "" {
		spec.UpdateStrategy = &cluster.UpdateStrategy{
			Type: string(k8sDs.Spec.UpdateStrategy.Type),
		}
		if k8sDs.Spec.UpdateStrategy.RollingUpdate != nil {
			if k8sDs.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable != nil {
				spec.UpdateStrategy.MaxUnavailable = k8sDs.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.String()
			}
			if k8sDs.Spec.UpdateStrategy.RollingUpdate.MaxSurge != nil {
				spec.UpdateStrategy.MaxSurge = k8sDs.Spec.UpdateStrategy.RollingUpdate.MaxSurge.String()
			}
		}
	}

	// Selector
	if k8sDs.Spec.Selector != nil {
		spec.Selector = &cluster.LabelSelector{
			MatchLabels: k8sDs.Spec.Selector.MatchLabels,
		}
	}

	return spec
}

// convertDaemonSetStatus 转换 DaemonSet Status
func convertDaemonSetStatus(k8sDs *appsv1.DaemonSet) cluster.DaemonSetStatus {
	status := cluster.DaemonSetStatus{
		ObservedGeneration:     k8sDs.Status.ObservedGeneration,
		DesiredNumberScheduled: k8sDs.Status.DesiredNumberScheduled,
		CurrentNumberScheduled: k8sDs.Status.CurrentNumberScheduled,
		NumberReady:            k8sDs.Status.NumberReady,
		NumberAvailable:        k8sDs.Status.NumberAvailable,
		NumberUnavailable:      k8sDs.Status.NumberUnavailable,
		NumberMisscheduled:     k8sDs.Status.NumberMisscheduled,
		UpdatedNumberScheduled: k8sDs.Status.UpdatedNumberScheduled,
		CollisionCount:         k8sDs.Status.CollisionCount,
	}

	// Conditions
	for _, c := range k8sDs.Status.Conditions {
		status.Conditions = append(status.Conditions, cluster.WorkloadCondition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
			LastTransitionTime: c.LastTransitionTime.Format(time.RFC3339),
		})
	}

	return status
}

// determineDaemonSetRollout 判断 DaemonSet 发布状态
func determineDaemonSetRollout(k8sDs *appsv1.DaemonSet) *cluster.WorkloadRollout {
	rollout := &cluster.WorkloadRollout{
		Phase:  "Complete",
		Badges: make([]string, 0),
	}

	desired := k8sDs.Status.DesiredNumberScheduled

	// 判断状态
	if k8sDs.Status.UpdatedNumberScheduled < desired {
		rollout.Phase = "Progressing"
		rollout.Message = "Rolling update in progress"
		rollout.Badges = append(rollout.Badges, "Updating")
	} else if k8sDs.Status.NumberReady < desired {
		rollout.Phase = "Progressing"
		rollout.Message = "Waiting for pods to be ready"
		rollout.Badges = append(rollout.Badges, "Scaling")
	} else if k8sDs.Status.NumberReady == 0 && desired > 0 {
		rollout.Phase = "Degraded"
		rollout.Message = "No pods are ready"
	}

	if k8sDs.Status.NumberMisscheduled > 0 {
		rollout.Badges = append(rollout.Badges, "Misscheduled")
	}

	if k8sDs.Status.NumberUnavailable > 0 {
		rollout.Badges = append(rollout.Badges, "Unavailable")
	}

	return rollout
}

// =============================================================================
// ReplicaSet 转换
// =============================================================================

// ConvertReplicaSet 转换 K8s ReplicaSet 到 model_v3
func ConvertReplicaSet(k8sRs *appsv1.ReplicaSet) cluster.ReplicaSet {
	rs := cluster.ReplicaSet{
		CommonMeta: buildCommonMeta(
			string(k8sRs.UID),
			k8sRs.Name,
			k8sRs.Namespace,
			"ReplicaSet",
			k8sRs.Labels,
			k8sRs.CreationTimestamp.Time,
		),
		Replicas:          *k8sRs.Spec.Replicas,
		ReadyReplicas:     k8sRs.Status.ReadyReplicas,
		AvailableReplicas: k8sRs.Status.AvailableReplicas,
	}

	// 设置 Owner
	if len(k8sRs.OwnerReferences) > 0 {
		owner := k8sRs.OwnerReferences[0]
		rs.OwnerKind = owner.Kind
		rs.OwnerName = owner.Name
	}

	if k8sRs.Spec.Selector != nil {
		rs.Selector = k8sRs.Spec.Selector.MatchLabels
	}

	return rs
}

// =============================================================================
// Job 转换
// =============================================================================

// ConvertJob 转换 K8s Job 到 model_v3
func ConvertJob(k8sJob *batchv1.Job) cluster.Job {
	job := cluster.Job{
		CommonMeta: buildCommonMeta(
			string(k8sJob.UID),
			k8sJob.Name,
			k8sJob.Namespace,
			"Job",
			k8sJob.Labels,
			k8sJob.CreationTimestamp.Time,
		),
		Active:    k8sJob.Status.Active,
		Succeeded: k8sJob.Status.Succeeded,
		Failed:    k8sJob.Status.Failed,
	}

	// 规格
	job.Completions = k8sJob.Spec.Completions
	job.Parallelism = k8sJob.Spec.Parallelism
	job.BackoffLimit = k8sJob.Spec.BackoffLimit

	// Pod 模板
	job.Template = convertPodTemplate(&k8sJob.Spec.Template.Spec)

	// Conditions
	for _, c := range k8sJob.Status.Conditions {
		job.Conditions = append(job.Conditions, cluster.WorkloadCondition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
			LastTransitionTime: c.LastTransitionTime.Format(time.RFC3339),
		})
	}

	// Owner
	if len(k8sJob.OwnerReferences) > 0 {
		owner := k8sJob.OwnerReferences[0]
		job.OwnerKind = owner.Kind
		job.OwnerName = owner.Name
	}

	if k8sJob.Status.StartTime != nil {
		t := k8sJob.Status.StartTime.Time
		job.StartTime = &t
	}
	if k8sJob.Status.CompletionTime != nil {
		t := k8sJob.Status.CompletionTime.Time
		job.FinishTime = &t
		job.Complete = true
	}

	return job
}

// =============================================================================
// CronJob 转换
// =============================================================================

// ConvertCronJob 转换 K8s CronJob 到 model_v3
func ConvertCronJob(k8sCj *batchv1.CronJob) cluster.CronJob {
	cj := cluster.CronJob{
		CommonMeta: buildCommonMeta(
			string(k8sCj.UID),
			k8sCj.Name,
			k8sCj.Namespace,
			"CronJob",
			k8sCj.Labels,
			k8sCj.CreationTimestamp.Time,
		),
		Schedule:          k8sCj.Spec.Schedule,
		Suspend:           k8sCj.Spec.Suspend != nil && *k8sCj.Spec.Suspend,
		ConcurrencyPolicy: string(k8sCj.Spec.ConcurrencyPolicy),
		ActiveJobs:        int32(len(k8sCj.Status.Active)),
	}

	// 历史保留限制
	cj.SuccessfulJobsHistoryLimit = k8sCj.Spec.SuccessfulJobsHistoryLimit
	cj.FailedJobsHistoryLimit = k8sCj.Spec.FailedJobsHistoryLimit

	// Pod 模板（从 JobTemplate.Spec.Template 提取）
	cj.Template = convertPodTemplate(&k8sCj.Spec.JobTemplate.Spec.Template.Spec)

	if k8sCj.Status.LastScheduleTime != nil {
		t := k8sCj.Status.LastScheduleTime.Time
		cj.LastScheduleTime = &t
	}
	if k8sCj.Status.LastSuccessfulTime != nil {
		t := k8sCj.Status.LastSuccessfulTime.Time
		cj.LastSuccessfulTime = &t
	}

	return cj
}

// =============================================================================
// PV/PVC 转换
// =============================================================================

// ConvertPersistentVolume 转换 K8s PV 到 model_v3
func ConvertPersistentVolume(k8sPv *corev1.PersistentVolume) cluster.PersistentVolume {
	pv := cluster.PersistentVolume{
		CommonMeta: buildCommonMeta(
			string(k8sPv.UID),
			k8sPv.Name,
			"",
			"PersistentVolume",
			k8sPv.Labels,
			k8sPv.CreationTimestamp.Time,
		),
		Capacity:         k8sPv.Spec.Capacity.Storage().String(),
		Phase:            string(k8sPv.Status.Phase),
		StorageClass:     k8sPv.Spec.StorageClassName,
		ReclaimPolicy:    string(k8sPv.Spec.PersistentVolumeReclaimPolicy),
		VolumeSourceType: detectVolumeSourceType(k8sPv.Spec.PersistentVolumeSource),
	}

	// ClaimRef
	if k8sPv.Spec.ClaimRef != nil {
		pv.ClaimRefName = k8sPv.Spec.ClaimRef.Name
		pv.ClaimRefNS = k8sPv.Spec.ClaimRef.Namespace
	}

	for _, mode := range k8sPv.Spec.AccessModes {
		pv.AccessModes = append(pv.AccessModes, string(mode))
	}

	return pv
}

// detectVolumeSourceType 检测 PV 卷来源类型
func detectVolumeSourceType(src corev1.PersistentVolumeSource) string {
	switch {
	case src.HostPath != nil:
		return "HostPath"
	case src.NFS != nil:
		return "NFS"
	case src.CSI != nil:
		return "CSI"
	case src.Local != nil:
		return "Local"
	case src.AWSElasticBlockStore != nil:
		return "AWSElasticBlockStore"
	case src.GCEPersistentDisk != nil:
		return "GCEPersistentDisk"
	case src.AzureDisk != nil:
		return "AzureDisk"
	case src.AzureFile != nil:
		return "AzureFile"
	case src.CephFS != nil:
		return "CephFS"
	case src.FC != nil:
		return "FC"
	case src.ISCSI != nil:
		return "iSCSI"
	case src.RBD != nil:
		return "RBD"
	default:
		return "Unknown"
	}
}

// ConvertPersistentVolumeClaim 转换 K8s PVC 到 model_v3
func ConvertPersistentVolumeClaim(k8sPvc *corev1.PersistentVolumeClaim) cluster.PersistentVolumeClaim {
	pvc := cluster.PersistentVolumeClaim{
		CommonMeta: buildCommonMeta(
			string(k8sPvc.UID),
			k8sPvc.Name,
			k8sPvc.Namespace,
			"PersistentVolumeClaim",
			k8sPvc.Labels,
			k8sPvc.CreationTimestamp.Time,
		),
		Phase:      string(k8sPvc.Status.Phase),
		VolumeName: k8sPvc.Spec.VolumeName,
	}

	if k8sPvc.Spec.StorageClassName != nil {
		pvc.StorageClass = *k8sPvc.Spec.StorageClassName
	}

	if req, ok := k8sPvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		pvc.RequestedCapacity = req.String()
	}
	if cap, ok := k8sPvc.Status.Capacity[corev1.ResourceStorage]; ok {
		pvc.ActualCapacity = cap.String()
	}

	// VolumeMode
	if k8sPvc.Spec.VolumeMode != nil {
		pvc.VolumeMode = string(*k8sPvc.Spec.VolumeMode)
	}

	for _, mode := range k8sPvc.Spec.AccessModes {
		pvc.AccessModes = append(pvc.AccessModes, string(mode))
	}

	return pvc
}

// =============================================================================
// 辅助函数
// =============================================================================

// buildCommonMeta 构建公共元数据
//
// 所有资源转换函数都调用此函数构建 CommonMeta。
//
// 注意: NodeName, OwnerKind, OwnerName 等关联字段
// 需要在各资源的转换函数中单独设置。
func buildCommonMeta(uid, name, namespace, kind string, labels map[string]string, createdAt interface{}) model_v3.CommonMeta {
	meta := model_v3.CommonMeta{
		UID:       uid,
		Name:      name,
		Namespace: namespace,
		Kind:      kind,
		Labels:    labels,
	}

	// 处理时间类型
	switch t := createdAt.(type) {
	case time.Time:
		meta.CreatedAt = t
	}

	return meta
}

// isPodReady 判断 Pod 是否 Ready
//
// 通过检查 Pod 的 Conditions 中是否存在 Ready=True 来判断。
// Pod Ready 意味着所有容器都已就绪，可以接收流量。
func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}
