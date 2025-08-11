package dataapi

import (
	"NeuroController/internal/ingest/store"
	model "NeuroController/model/metrics"
	"time"
)

// MetricsStoreAPI 提供对 metrics store 的只读访问接口
type MetricsStoreAPI struct {
    st *store.Store
}

// NewMetricsStoreAPI 构造函数，注入底层 store
func NewMetricsStoreAPI(st *store.Store) *MetricsStoreAPI {
    return &MetricsStoreAPI{st: st}
}

// GetLatest 简单调用底层 GetLatest
func (api *MetricsStoreAPI) GetLatest(node string) *model.NodeMetricsSnapshot {
    return api.st.GetLatest(node)
}

// Range 简单调用底层 Range
func (api *MetricsStoreAPI) Range(node string, since, until time.Time) []*model.NodeMetricsSnapshot {
    return api.st.Range(node, since, until)
}

// DumpAll 简单调用底层 DumpAll
func (api *MetricsStoreAPI) DumpAll() map[string][]*model.NodeMetricsSnapshot {
    return api.st.DumpAll()
}