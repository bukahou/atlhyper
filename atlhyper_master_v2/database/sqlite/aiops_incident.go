// atlhyper_master_v2/database/sqlite/aiops_incident.go
// AIOps 事件 SQLite 方言实现
package sqlite

import (
	"AtlHyper/atlhyper_master_v2/aiops/severity"
	"database/sql"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// aIOpsIncidentDialect AIOps 事件 SQLite 方言
type aIOpsIncidentDialect struct{}

// incidentColumns 事件表的列，顺序与 ScanIncident 一致
var incidentColumns = []string{
	"id", "cluster_id", "state", "severity", "subject", "root_cause", "peak_risk",
	"started_at", "escalated_at", "resolved_at", "close_reason", "duration_s", "recurrence", "summary", "created_at",
}

func (d *aIOpsIncidentDialect) IncidentColumns(prefix string) string {
	return prefix + strings.Join(incidentColumns, ", "+prefix)
}

func (d *aIOpsIncidentDialect) InsertIncident(inc *database.AIOpsIncident) (string, []any) {
	return `INSERT INTO aiops_incidents (id, cluster_id, state, severity, subject, root_cause, peak_risk, started_at, summary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{inc.ID, inc.ClusterID, inc.State, inc.Severity, inc.Subject, inc.RootCause, inc.PeakRisk,
			utcStamp(inc.StartedAt), inc.Summary, utcStamp(inc.CreatedAt)}
}

func (d *aIOpsIncidentDialect) SelectByID(id string) (string, []any) {
	return `SELECT ` + d.IncidentColumns("") + ` FROM aiops_incidents WHERE id = ?`, []any{id}
}

func (d *aIOpsIncidentDialect) UpdateState(id, state string) (string, []any) {
	return `UPDATE aiops_incidents SET state = ? WHERE id = ?`, []any{state, id}
}

func (d *aIOpsIncidentDialect) RaiseSeverity(id, newSeverity string, peakRisk float64) (string, []any) {
	return `UPDATE aiops_incidents SET
			severity = CASE WHEN ` + severity.RankSQL("?") + ` > ` + severity.RankSQL("severity") + ` THEN ? ELSE severity END,
			peak_risk = MAX(peak_risk, ?)
		WHERE id = ?`, []any{newSeverity, newSeverity, peakRisk, id}
}

func (d *aIOpsIncidentDialect) MarkEscalated(id string, escalatedAt time.Time) (string, []any) {
	return `UPDATE aiops_incidents SET escalated_at = ? WHERE id = ? AND escalated_at IS NULL`,
		[]any{utcStamp(escalatedAt), id}
}

// SetAnomalyDuration 时长按整秒相减：julianday 相减再乘 86400 是浮点运算，截断后会少 1 秒
func (d *aIOpsIncidentDialect) SetAnomalyDuration(id string, endedAt time.Time) (string, []any) {
	return `UPDATE aiops_incidents SET
			duration_s = CAST(strftime('%s', ?) AS INTEGER) - CAST(strftime('%s', started_at) AS INTEGER)
		WHERE id = ?`,
		[]any{utcStamp(endedAt), id}
}

// Resolve 时长是「异常持续了多久」，不含恢复观察期：
// 从观察期关闭的事件沿用进入观察期时记下的值（CASE 里读到的是更新前的 state）；
// 其余（失联关闭时还在警告 / 事件中）算到关闭时刻。
func (d *aIOpsIncidentDialect) Resolve(id string, resolvedAt time.Time, reason string) (string, []any) {
	return `UPDATE aiops_incidents SET
			duration_s = CASE WHEN state = 'recovery' AND duration_s > 0 THEN duration_s
				ELSE CAST(strftime('%s', ?) AS INTEGER) - CAST(strftime('%s', started_at) AS INTEGER) END,
			state = 'stable', resolved_at = ?, close_reason = ?
		WHERE id = ?`,
		[]any{utcStamp(resolvedAt), utcStamp(resolvedAt), reason, id}
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
		[]any{entry.IncidentID, utcStamp(entry.Timestamp), entry.EventType, entry.EntityKey, entry.Detail}
}

func (d *aIOpsIncidentDialect) SelectTimeline(incidentID string) (string, []any) {
	return `SELECT id, incident_id, timestamp, event_type, entity_key, detail
		FROM aiops_incident_timeline WHERE incident_id = ? ORDER BY timestamp ASC`, []any{incidentID}
}

func (d *aIOpsIncidentDialect) UpsertSignal(signal *database.AIOpsIncidentSignal) (string, []any) {
	return `INSERT INTO aiops_incident_signals
			(incident_id, entity_key, metric, first_seen, last_seen, hits, max_score, detail)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT(incident_id, entity_key, metric) DO UPDATE SET
			last_seen = excluded.last_seen,
			hits = aiops_incident_signals.hits + 1,
			max_score = MAX(aiops_incident_signals.max_score, excluded.max_score),
			detail = excluded.detail`,
		[]any{signal.IncidentID, signal.EntityKey, signal.Metric,
			utcStamp(signal.LastSeen), utcStamp(signal.LastSeen),
			signal.MaxScore, signal.Detail}
}

func (d *aIOpsIncidentDialect) SelectSignals(incidentID string) (string, []any) {
	return `SELECT incident_id, entity_key, metric, first_seen, last_seen, hits, max_score, detail
		FROM aiops_incident_signals WHERE incident_id = ? ORDER BY max_score DESC, metric, entity_key`, []any{incidentID}
}

func (d *aIOpsIncidentDialect) ScanSignal(rows *sql.Rows) (*database.AIOpsIncidentSignal, error) {
	s := &database.AIOpsIncidentSignal{}
	var firstSeen, lastSeen string
	if err := rows.Scan(&s.IncidentID, &s.EntityKey, &s.Metric, &firstSeen, &lastSeen, &s.Hits, &s.MaxScore, &s.Detail); err != nil {
		return nil, err
	}
	s.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
	s.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	return s, nil
}

func (d *aIOpsIncidentDialect) SaveLink(link *database.AIOpsIncidentLink) (string, []any) {
	return `INSERT INTO aiops_incident_links (incident_id, cluster_id, root_incident_id, path, linked_at, unlinked_at)
		VALUES (?, ?, ?, ?, ?, NULL)
		ON CONFLICT(incident_id) DO UPDATE SET
			cluster_id = excluded.cluster_id,
			root_incident_id = excluded.root_incident_id,
			path = excluded.path,
			linked_at = excluded.linked_at,
			unlinked_at = NULL`,
		[]any{link.IncidentID, link.ClusterID, link.RootIncidentID, link.Path, utcStamp(link.LinkedAt)}
}

func (d *aIOpsIncidentDialect) UnlinkIncident(incidentID string, at time.Time) (string, []any) {
	return `UPDATE aiops_incident_links SET unlinked_at = ? WHERE incident_id = ? AND unlinked_at IS NULL`,
		[]any{utcStamp(at), incidentID}
}

func (d *aIOpsIncidentDialect) SelectActiveLinks(clusterID string) (string, []any) {
	query := `SELECT incident_id, cluster_id, root_incident_id, path, linked_at, unlinked_at
		FROM aiops_incident_links WHERE unlinked_at IS NULL`
	if clusterID == "" {
		return query + ` ORDER BY linked_at`, nil
	}
	return query + ` AND cluster_id = ? ORDER BY linked_at`, []any{clusterID}
}

func (d *aIOpsIncidentDialect) ScanLink(rows *sql.Rows) (*database.AIOpsIncidentLink, error) {
	link := &database.AIOpsIncidentLink{}
	var linkedAt string
	var unlinkedAt sql.NullString
	if err := rows.Scan(&link.IncidentID, &link.ClusterID, &link.RootIncidentID, &link.Path, &linkedAt, &unlinkedAt); err != nil {
		return nil, err
	}
	link.LinkedAt, _ = time.Parse(time.RFC3339, linkedAt)
	if unlinkedAt.Valid {
		t, _ := time.Parse(time.RFC3339, unlinkedAt.String)
		link.UnlinkedAt = &t
	}
	return link, nil
}

func (d *aIOpsIncidentDialect) ScanIncident(rows *sql.Rows) (*database.AIOpsIncident, error) {
	inc := &database.AIOpsIncident{}
	var startedAt, createdAt string
	var escalatedAt, resolvedAt, rootCause, summary sql.NullString
	err := rows.Scan(&inc.ID, &inc.ClusterID, &inc.State, &inc.Severity, &inc.Subject, &rootCause, &inc.PeakRisk,
		&startedAt, &escalatedAt, &resolvedAt, &inc.CloseReason, &inc.DurationS, &inc.Recurrence, &summary, &createdAt)
	if err != nil {
		return nil, err
	}
	inc.RootCause, inc.Summary = rootCause.String, summary.String
	inc.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	inc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if escalatedAt.Valid {
		t, _ := time.Parse(time.RFC3339, escalatedAt.String)
		inc.EscalatedAt = &t
	}
	if resolvedAt.Valid {
		t, _ := time.Parse(time.RFC3339, resolvedAt.String)
		inc.ResolvedAt = &t
	}
	// 时长是「异常持续了多久」：仍在异常中的事件按当前时刻动态算；
	// 恢复观察中的事件用进入观察期时记下的值（观察期不算异常时长）
	if inc.ResolvedAt == nil && !(inc.State == "recovery" && inc.DurationS > 0) {
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

func (d *aIOpsIncidentDialect) DeleteResolvedBefore(before time.Time) (string, []any) {
	return "DELETE FROM aiops_incidents WHERE resolved_at IS NOT NULL AND resolved_at < ?", []any{utcStamp(before)}
}
