// Package handler Gateway HTTP 处理器
//
// node_metrics.go - 节点指标 API 处理器
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v2"
)

// NodeMetricsHandler 节点指标处理器
type NodeMetricsHandler struct {
	metricsRepo database.NodeMetricsRepository
}

// NewNodeMetricsHandler 创建节点指标处理器
func NewNodeMetricsHandler(metricsRepo database.NodeMetricsRepository) *NodeMetricsHandler {
	return &NodeMetricsHandler{
		metricsRepo: metricsRepo,
	}
}

// Route 路由分发
// 处理 /api/v2/node-metrics/* 路径
func (h *NodeMetricsHandler) Route(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 解析路径: /api/v2/node-metrics/{nodeName}[/history]
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/node-metrics")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		// /api/v2/node-metrics 或 /api/v2/node-metrics/ -> 列表
		h.List(w, r)
		return
	}

	// 检查是否有 /history 后缀
	if strings.HasSuffix(path, "/history") {
		nodeName := strings.TrimSuffix(path, "/history")
		h.getHistory(w, r, nodeName)
		return
	}

	// /api/v2/node-metrics/{nodeName} -> 详情
	h.getDetail(w, r, path)
}

// List 获取集群所有节点指标
// GET /api/v2/node-metrics?cluster_id=xxx
func (h *NodeMetricsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 获取所有节点实时数据
	latestList, err := h.metricsRepo.ListLatest(r.Context(), clusterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 解析 JSON 并构建响应
	nodes := make([]*model_v2.NodeMetricsSnapshot, 0, len(latestList))
	for _, latest := range latestList {
		var snapshot model_v2.NodeMetricsSnapshot
		if err := json.Unmarshal([]byte(latest.SnapshotJSON), &snapshot); err != nil {
			continue
		}
		nodes = append(nodes, &snapshot)
	}

	// 计算汇总统计
	summary := h.calculateSummary(nodes)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"summary": summary,
		"nodes":   nodes,
	})
}

// getDetail 获取单节点详情
func (h *NodeMetricsHandler) getDetail(w http.ResponseWriter, r *http.Request, nodeName string) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	latest, err := h.metricsRepo.GetLatest(r.Context(), clusterID, nodeName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if latest == nil {
		writeError(w, http.StatusNotFound, "node metrics not found")
		return
	}

	var snapshot model_v2.NodeMetricsSnapshot
	if err := json.Unmarshal([]byte(latest.SnapshotJSON), &snapshot); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse snapshot")
		return
	}

	writeJSON(w, http.StatusOK, snapshot)
}

// getHistory 获取节点历史数据
func (h *NodeMetricsHandler) getHistory(w http.ResponseWriter, r *http.Request, nodeName string) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 解析时间范围
	hours := 24
	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
			hours = h
		}
	}

	end := time.Now()
	start := end.Add(-time.Duration(hours) * time.Hour)

	history, err := h.metricsRepo.GetHistory(r.Context(), clusterID, nodeName, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 转换为 MetricsDataPoint
	dataPoints := make([]model_v2.MetricsDataPoint, 0, len(history))
	for _, h := range history {
		dataPoints = append(dataPoints, model_v2.MetricsDataPoint{
			Timestamp:   h.Timestamp,
			NodeName:    h.NodeName,
			CPUUsage:    h.CPUUsage,
			MemoryUsage: h.MemoryUsage,
			DiskUsage:   h.DiskUsage,
			DiskIORead:  h.DiskIORead,
			DiskIOWrite: h.DiskIOWrite,
			NetworkRx:   h.NetworkRx,
			NetworkTx:   h.NetworkTx,
			CPUTemp:     h.CPUTemp,
			Load1:       h.Load1,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node_name": nodeName,
		"start":     start,
		"end":       end,
		"data":      dataPoints,
	})
}

// calculateSummary 计算集群汇总统计
func (h *NodeMetricsHandler) calculateSummary(nodes []*model_v2.NodeMetricsSnapshot) model_v2.ClusterMetricsSummary {
	summary := model_v2.ClusterMetricsSummary{
		TotalNodes:  len(nodes),
		OnlineNodes: len(nodes),
	}

	if len(nodes) == 0 {
		return summary
	}

	var (
		totalCPU, totalMem, totalDisk float64
		totalMemBytes, usedMemBytes   int64
		totalDiskBytes, usedDiskBytes int64
		totalNetRx, totalNetTx        float64
		maxCPU, maxMem, maxDisk       float64
		totalTemp                     float64
		tempCount                     int
	)

	for _, node := range nodes {
		// CPU
		totalCPU += node.CPU.UsagePercent
		if node.CPU.UsagePercent > maxCPU {
			maxCPU = node.CPU.UsagePercent
		}

		// 内存
		totalMem += node.Memory.UsagePercent
		totalMemBytes += node.Memory.Total
		usedMemBytes += node.Memory.Used
		if node.Memory.UsagePercent > maxMem {
			maxMem = node.Memory.UsagePercent
		}

		// 磁盘
		if disk := node.GetPrimaryDisk(); disk != nil {
			totalDisk += disk.UsagePercent
			totalDiskBytes += disk.Total
			usedDiskBytes += disk.Used
			if disk.UsagePercent > maxDisk {
				maxDisk = disk.UsagePercent
			}
		}

		// 网络
		if net := node.GetPrimaryNetwork(); net != nil {
			totalNetRx += net.RxRate
			totalNetTx += net.TxRate
		}

		// 温度
		if node.Temperature.CPUTemp > 0 {
			totalTemp += node.Temperature.CPUTemp
			tempCount++
			if node.Temperature.CPUTemp > summary.MaxCPUTemp {
				summary.MaxCPUTemp = node.Temperature.CPUTemp
			}
		}
	}

	n := float64(len(nodes))
	summary.AvgCPUUsage = totalCPU / n
	summary.AvgMemoryUsage = totalMem / n
	summary.AvgDiskUsage = totalDisk / n
	summary.MaxCPUUsage = maxCPU
	summary.MaxMemoryUsage = maxMem
	summary.MaxDiskUsage = maxDisk
	summary.TotalMemory = totalMemBytes
	summary.UsedMemory = usedMemBytes
	summary.TotalDisk = totalDiskBytes
	summary.UsedDisk = usedDiskBytes
	summary.TotalNetworkRx = int64(totalNetRx)
	summary.TotalNetworkTx = int64(totalNetTx)

	if tempCount > 0 {
		summary.AvgCPUTemp = totalTemp / float64(tempCount)
	}

	return summary
}
