// atlhyper_master_v2/database/sqlite/aiops_incident.go
// AIOps 事件 SQLite 方言实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// aIOpsIncidentDialect AIOps 事件 SQLite 方言
type aIOpsIncidentDialect struct{}

func (d *aIOpsIncidentDialect) InsertIncident(inc *database.AIOpsIncident) (string, []any) {
	return `INSERT INTO aiops_incidents (id, cluster_id, state, severity, root_cause, peak_risk, started_at, resolved_at, duration_s, recurrence, summary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{inc.ID, inc.ClusterID, inc.State, inc.Severity, inc.RootCause, inc.PeakRisk,
			inc.StartedAt.Format(time.RFC3339), nil, 0, 0, inc.Summary, inc.CreatedAt.Format(time.RFC3339)}
}

func (d *aIOpsIncidentDialect) SelectByID(id string) (string, []any) {
	return `SELECT id, cluster_id, state, severity, root_cause, peak_risk, started_at, resolved_at, duration_s, recurrence, summary, created_at
		FROM aiops_incidents WHERE id = ?`, []any{id}
}

func (d *aIOpsIncidentDialect) UpdateState(id, state, severity string) (string, []any) {
	return `UPDATE aiops_incidents SET state = ?, severity = ? WHERE id = ?`, []any{state, severity, id}
}

func (d *aIOpsIncidentDialect) Resolve(id string, resolvedAt time.Time) (string, []any) {
	return `UPDATE aiops_incidents SET state = 'stable', resolved_at = ?, duration_s = CAST((julianday(?) - julianday(started_at)) * 86400 AS INTEGER) WHERE id = ?`,
		[]any{resolvedAt.Format(time.RFC3339), resolvedAt.Format(time.RFC3339), id}
}

func (d *aIOpsIncidentDialect) UpdateRootCause(id, rootCause string) (string, []any) {
	return `UPDATE aiops_incidents SET root_cause = ? WHERE id = ?`, []any{rootCause, id}
}

func (d *aIOpsIncidentDialect) UpdatePeakRisk(id string, peakRisk float64) (string, []any) {
	return `UPDATE aiops_incidents SET peak_risk = MAX(peak_risk, ?) WHERE id = ?`, []any{peakRisk, id}
}

func (d *aIOpsIncidentDialect) IncrementRecurrence(id string) (string, []any) {
	return `UPDATE aiops_incidents SET recurrence = recurrence + 1 WHERE id = ?`, []any{id}
}

func (d *aIOpsIncidentDialect) InsertEntity(entity *database.AIOpsIncidentEntity) (string, []any) {
	return `INSERT OR REPLACE INTO aiops_incident_entities (incident_id, entity_key, entity_type, r_local, r_final, role)
		VALUES (?, ?, ?, ?, ?, ?)`,
		[]any{entity.IncidentID, entity.EntityKey, entity.EntityType, entity.RLocal, entity.RFinal, entity.Role}
}

func (d *aIOpsIncidentDialect) SelectEntities(incidentID string) (string, []any) {
	return `SELECT incident_id, entity_key, entity_type, r_local, r_final, role
		FROM aiops_incident_entities WHERE incident_id = ?`, []any{incidentID}
}

func (d *aIOpsIncidentDialect) InsertTimeline(entry *database.AIOpsIncidentTimeline) (string, []any) {
	return `INSERT INTO aiops_incident_timeline (incident_id, timestamp, event_type, entity_key, detail)
		VALUES (?, ?, ?, ?, ?)`,
		[]any{entry.IncidentID, entry.Timestamp.Format(time.RFC3339), entry.EventType, entry.EntityKey, entry.Detail}
}

func (d *aIOpsIncidentDialect) SelectTimeline(incidentID string) (string, []any) {
	return `SELECT id, incident_id, timestamp, event_type, entity_key, detail
		FROM aiops_incident_timeline WHERE incident_id = ? ORDER BY timestamp ASC`, []any{incidentID}
}

func (d *aIOpsIncidentDialect) ScanIncident(rows *sql.Rows) (*database.AIOpsIncident, error) {
	inc := &database.AIOpsIncident{}
	var startedAt, createdAt string
	var resolvedAt sql.NullString
	err := rows.Scan(&inc.ID, &inc.ClusterID, &inc.State, &inc.Severity, &inc.RootCause, &inc.PeakRisk,
		&startedAt, &resolvedAt, &inc.DurationS, &inc.Recurrence, &inc.Summary, &createdAt)
	if err != nil {
		return nil, err
	}
	inc.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	inc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if resolvedAt.Valid {
		t, _ := time.Parse(time.RFC3339, resolvedAt.String)
		inc.ResolvedAt = &t
	}
	// 活跃事件：动态计算持续时间（DB 中只有 Resolve 时才写入 duration_s）
	if inc.ResolvedAt == nil {
		inc.DurationS = int64(time.Since(inc.StartedAt).Seconds())
	}
	return inc, nil
}

func (d *aIOpsIncidentDialect) ScanEntity(rows *sql.Rows) (*database.AIOpsIncidentEntity, error) {
	e := &database.AIOpsIncidentEntity{}
	err := rows.Scan(&e.IncidentID, &e.EntityKey, &e.EntityType, &e.RLocal, &e.RFinal, &e.Role)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (d *aIOpsIncidentDialect) ScanTimeline(rows *sql.Rows) (*database.AIOpsIncidentTimeline, error) {
	t := &database.AIOpsIncidentTimeline{}
	var timestamp string
	err := rows.Scan(&t.ID, &t.IncidentID, &timestamp, &t.EventType, &t.EntityKey, &t.Detail)
	if err != nil {
		return nil, err
	}
	t.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
	return t, nil
}
