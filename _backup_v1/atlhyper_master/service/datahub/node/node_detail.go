// atlhyper_master/service/node/node_detail.go
package node

import (
	"context"
	"fmt"
	"strings"

	"AtlHyper/atlhyper_master/model/dto"
	"AtlHyper/atlhyper_master/repository"
	mod "AtlHyper/model/k8s"
)

// GetNodeDetail —— 根据 clusterID + nodeName 返回单个节点的扁平详情 DTO
func GetNodeDetail(ctx context.Context, clusterID, nodeName string) (*dto.NodeDetailDTO, error) {
	nodes, err := repository.Mem.GetNodeListLatest(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("get node list failed: %w", err)
	}
	for _, n := range nodes {
		if n.Summary.Name == nodeName {
			dto := fromModelToDetail(n)
			return &dto, nil
		}
	}
	return nil, fmt.Errorf("node not found: %s (cluster=%s)", nodeName, clusterID)
}

// fromModelToDetail —— Store 模型 → 详情 DTO（扁平化）
func fromModelToDetail(n mod.Node) dto.NodeDetailDTO {
	out := dto.NodeDetailDTO{
		// summary
		Name:        n.Summary.Name,
		Roles:       n.Summary.Roles,
		Ready:       strings.EqualFold(n.Summary.Ready, "true"),
		Schedulable: n.Summary.Schedulable,
		Age:         n.Summary.Age,
		CreatedAt:   n.Summary.CreationTime,
		Badges:      n.Summary.Badges,
		Reason:      n.Summary.Reason,
		Message:     n.Summary.Message,

		// addresses
		Hostname:   n.Addresses.Hostname,
		InternalIP: n.Addresses.InternalIP,
		ExternalIP: n.Addresses.ExternalIP,

		// info
		OSImage:      n.Info.OSImage,
		OS:           n.Info.OperatingSystem,
		Architecture: n.Info.Architecture,
		Kernel:       n.Info.KernelVersion,
		CRI:          n.Info.ContainerRuntimeVersion,
		Kubelet:      n.Info.KubeletVersion,
		KubeProxy:    n.Info.KubeProxyVersion,

		// spec
		PodCIDRs:   n.Spec.PodCIDRs,
		ProviderID: n.Spec.ProviderID,

		// labels
		Labels: n.Labels,
	}

	// capacity/allocatable
	out.CPUCapacityCores = parseCPUToInt(n.Capacity.CPU)
	out.CPUAllocatableCores = parseCPUToInt(n.Allocatable.CPU)
	out.MemCapacityGiB = parseMemToGiB(n.Capacity.Memory)
	out.MemAllocatableGiB = parseMemToGiB(n.Allocatable.Memory)
	out.PodsCapacity = atoiSafe(n.Capacity.Pods)
	out.PodsAllocatable = atoiSafe(n.Allocatable.Pods)
	out.EphemeralStorageGiB = parseMemToGiB(n.Capacity.EphemeralStorage)

	// metrics（若有）
	if n.Metrics != nil {
		out.CPUUsageCores = parseCPUUsageToCores(n.Metrics.CPU.Usage)
		out.CPUUtilPct = n.Metrics.CPU.UtilPct
		out.MemUsageGiB = parseMemToGiB(n.Metrics.Memory.Usage)
		out.MemUtilPct = n.Metrics.Memory.UtilPct
		out.PodsUsed = n.Metrics.Pods.Used
		out.PodsUtilPct = n.Metrics.Pods.UtilPct
		out.PressureMemory = n.Metrics.Pressure.MemoryPressure
		out.PressureDisk = n.Metrics.Pressure.DiskPressure
		out.PressurePID = n.Metrics.Pressure.PIDPressure
		out.NetworkUnavailable = n.Metrics.Pressure.NetworkUnavailable
	}

	// conditions
	for _, c := range n.Conditions {
		out.Conditions = append(out.Conditions, dto.NodeCondDTO{
			Type:      c.Type,
			Status:    c.Status,
			Reason:    c.Reason,
			Message:   c.Message,
			Heartbeat: c.LastHeartbeatTime,
			ChangedAt: c.LastTransitionTime,
		})
	}

	// taints
	for _, t := range n.Taints {
		out.Taints = append(out.Taints, dto.NodeTaintDTO{
			Key: t.Key, Value: t.Value, Effect: t.Effect,
		})
	}

	return out
}
