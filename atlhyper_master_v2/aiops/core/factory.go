// atlhyper_master_v2/aiops/core/factory.go
// AIOps 引擎工厂函数
package core

import (
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/baseline"
	"AtlHyper/atlhyper_master_v2/aiops/correlator"
	"AtlHyper/atlhyper_master_v2/aiops/incident"
	"AtlHyper/atlhyper_master_v2/aiops/risk"
	"AtlHyper/atlhyper_master_v2/aiops/statemachine"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
)

// EngineConfig AIOps 引擎配置
type EngineConfig struct {
	Store         datahub.Store
	GraphRepo     database.AIOpsGraphRepository
	BaselineRepo  database.AIOpsBaselineRepository
	IncidentRepo  database.AIOpsIncidentRepository
	SLORepo       database.SLORepository
	FlushInterval time.Duration
}

// NewEngine 创建 AIOps 引擎
func NewEngine(cfg EngineConfig) aiops.Engine {
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Minute
	}

	incStore := incident.NewStore(cfg.IncidentRepo)

	e := &engine{
		store:         cfg.Store,
		corr:          correlator.NewCorrelator(),
		stateManager:  baseline.NewStateManager(cfg.BaselineRepo),
		scorer:        risk.NewScorer(nil),
		incidentStore: incStore,
		graphRepo:     cfg.GraphRepo,
		sloRepo:       cfg.SLORepo,
		anomalyCache:  make(map[string][]*aiops.AnomalyResult),
		flushInterval: cfg.FlushInterval,
	}

	// 创建状态机，engine 本身作为 TransitionCallback
	e.sm = statemachine.NewStateMachine(e)

	return e
}
