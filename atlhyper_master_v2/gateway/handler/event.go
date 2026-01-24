// atlhyper_master_v2/gateway/handler/event.go
// Event 查询 API Handler
// 实时 Events 通过 Query 层查询 DataHub
// 历史 Events 通过 Database 查询
package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repository"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service"
)

// EventHandler Event Handler
type EventHandler struct {
	svc      service.Query
	database database.Database
}

// NewEventHandler 创建 EventHandler
func NewEventHandler(svc service.Query, db database.Database) *EventHandler {
	return &EventHandler{
		svc:      svc,
		database: db,
	}
}

// List 列出事件
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 解析查询参数
	params := r.URL.Query()
	clusterID := params.Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id required")
		return
	}

	source := params.Get("source") // realtime / history，默认 realtime

	if source == "history" {
		// 历史数据从 Database 查询
		h.listFromDatabase(w, r, clusterID)
	} else {
		// 实时数据从 Query 层查询
		h.listFromQuery(w, r, clusterID)
	}
}

// listFromQuery 从 Query 层查询实时 Events
func (h *EventHandler) listFromQuery(w http.ResponseWriter, r *http.Request, clusterID string) {
	params := r.URL.Query()

	opts := model.EventQueryOpts{}

	// 类型筛选
	if eventType := params.Get("type"); eventType != "" {
		opts.Type = eventType
	}

	// 时间范围
	if sinceStr := params.Get("since"); sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			opts.Since = since
		}
	}

	// 分页
	if limitStr := params.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	if offsetStr := params.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			opts.Offset = offset
		}
	}

	events, err := h.svc.GetEvents(r.Context(), clusterID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query events")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  len(events),
		"source": "realtime",
	})
}

// listFromDatabase 从 Database 查询历史 Events
func (h *EventHandler) listFromDatabase(w http.ResponseWriter, r *http.Request, clusterID string) {
	params := r.URL.Query()

	opts := repository.EventQueryOpts{}

	// 分页
	if limitStr := params.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	if offsetStr := params.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			opts.Offset = offset
		}
	}

	// 时间范围
	if sinceStr := params.Get("since"); sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			opts.Since = since
		}
	}

	// 类型筛选
	if eventType := params.Get("type"); eventType != "" {
		opts.Type = eventType
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	dbEvents, err := h.database.ClusterEventRepository().ListByCluster(ctx, clusterID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query events")
		return
	}

	total, _ := h.database.ClusterEventRepository().CountByCluster(ctx, clusterID)

	// 转换为前端期望的格式
	events := make([]map[string]interface{}, 0, len(dbEvents))
	for _, e := range dbEvents {
		events = append(events, map[string]interface{}{
			"uid":             e.DedupKey,
			"name":            e.Name,
			"namespace":       e.Namespace,
			"kind":            "Event",
			"created_at":      e.CreatedAt.Format(time.RFC3339),
			"type":            e.Type,
			"reason":          e.Reason,
			"message":         e.Message,
			"source":          e.SourceComponent,
			"involved_object": map[string]string{
				"kind":      e.InvolvedKind,
				"namespace": e.InvolvedNamespace,
				"name":      e.InvolvedName,
			},
			"count":           e.Count,
			"first_timestamp": e.FirstTimestamp.Format(time.RFC3339),
			"last_timestamp":  e.LastTimestamp.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
		"source": "history",
		"limit":  opts.Limit,
		"offset": opts.Offset,
	})
}

// ListByResource 按资源查询事件
func (h *EventHandler) ListByResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	params := r.URL.Query()
	clusterID := params.Get("cluster_id")
	kind := params.Get("kind")
	namespace := params.Get("namespace")
	name := params.Get("name")

	if clusterID == "" || kind == "" || name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, kind, name required")
		return
	}

	// 从 Query 层查询实时数据
	events, err := h.svc.GetEventsByResource(r.Context(), clusterID, kind, namespace, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query events")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  len(events),
	})
}
