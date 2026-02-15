// atlhyper_master_v2/aiops/statemachine/suppressor.go
// 告警抑制逻辑
package statemachine

import "AtlHyper/atlhyper_master_v2/aiops"

// ShouldSuppress 判断是否应该抑制告警
// 同一实体在 Incident/Recovery 状态期间不创建新 Incident
func (sm *StateMachine) ShouldSuppress(entityKey string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	entry, ok := sm.entries[entityKey]
	if !ok {
		return false
	}

	return entry.CurrentState == aiops.StateIncident || entry.CurrentState == aiops.StateRecovery
}

// GetActiveIncidentID 获取实体当前关联的 Incident ID
func (sm *StateMachine) GetActiveIncidentID(entityKey string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	entry, ok := sm.entries[entityKey]
	if !ok {
		return ""
	}
	return entry.IncidentID
}
