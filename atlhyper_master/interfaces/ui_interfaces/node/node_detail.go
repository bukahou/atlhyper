// atlhyper_master/interfaces/ui_interfaces/node/node_detail.go
package node

import (
	"context"
	"fmt"
	"strings"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/node"
)

// GetNodeDetail —— 根据 clusterID + nodeName 返回单个节点的扁平详情 DTO
func GetNodeDetail(ctx context.Context, clusterID, nodeName string) (*NodeDetailDTO, error) {
	nodes, err := datasource.GetNodeListLatest(ctx, clusterID)
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
func fromModelToDetail(n mod.Node) NodeDetailDTO {
	dto := NodeDetailDTO{
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
	dto.CPUCapacityCores = parseCPUToInt(n.Capacity.CPU)
	dto.CPUAllocatableCores = parseCPUToInt(n.Allocatable.CPU)
	dto.MemCapacityGiB = parseMemToGiB(n.Capacity.Memory)
	dto.MemAllocatableGiB = parseMemToGiB(n.Allocatable.Memory)
	dto.PodsCapacity = atoiSafe(n.Capacity.Pods)
	dto.PodsAllocatable = atoiSafe(n.Allocatable.Pods)
	dto.EphemeralStorageGiB = parseMemToGiB(n.Capacity.EphemeralStorage)

	// metrics（若有）
	if n.Metrics != nil {
		dto.CPUUsageCores = parseCPUUsageToCores(n.Metrics.CPU.Usage)
		dto.CPUUtilPct = n.Metrics.CPU.UtilPct
		dto.MemUsageGiB = parseMemToGiB(n.Metrics.Memory.Usage)
		dto.MemUtilPct = n.Metrics.Memory.UtilPct
		dto.PodsUsed = n.Metrics.Pods.Used
		dto.PodsUtilPct = n.Metrics.Pods.UtilPct
		dto.PressureMemory = n.Metrics.Pressure.MemoryPressure
		dto.PressureDisk = n.Metrics.Pressure.DiskPressure
		dto.PressurePID = n.Metrics.Pressure.PIDPressure
		dto.NetworkUnavailable = n.Metrics.Pressure.NetworkUnavailable
	}

	// conditions
	for _, c := range n.Conditions {
		dto.Conditions = append(dto.Conditions, NodeCondDTO{
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
		dto.Taints = append(dto.Taints, TaintDTO{
			Key: t.Key, Value: t.Value, Effect: t.Effect,
		})
	}

	return dto
}
