// atlhyper_master_v2/database/sqlite/notify_policy.go
// SQLite NotifyPolicyDialect 实现（单行表，id 恒为 1）
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyPolicyDialect struct{}

const notifyPolicyColumns = `incident_alerts_enabled, min_severity, notify_resolved, warning_flap_threshold,
	group_wait_seconds, group_interval_seconds, repeat_interval_seconds, k8s_event_alerts_enabled,
	default_receiver_ids, updated_at`

func (d *notifyPolicyDialect) Select() (string, []any) {
	return "SELECT " + notifyPolicyColumns + " FROM notify_policy WHERE id = 1", nil
}

func (d *notifyPolicyDialect) Upsert(policy *database.NotifyPolicy, now time.Time) (string, []any) {
	query := `INSERT INTO notify_policy (id, ` + notifyPolicyColumns + `)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			incident_alerts_enabled = excluded.incident_alerts_enabled,
			min_severity = excluded.min_severity,
			notify_resolved = excluded.notify_resolved,
			warning_flap_threshold = excluded.warning_flap_threshold,
			group_wait_seconds = excluded.group_wait_seconds,
			group_interval_seconds = excluded.group_interval_seconds,
			repeat_interval_seconds = excluded.repeat_interval_seconds,
			k8s_event_alerts_enabled = excluded.k8s_event_alerts_enabled,
			default_receiver_ids = excluded.default_receiver_ids,
			updated_at = excluded.updated_at`
	receiverIDs := policy.DefaultReceiverIDs
	if receiverIDs == nil {
		receiverIDs = []int64{}
	}
	return query, []any{
		boolToInt(policy.IncidentAlertsEnabled), policy.MinSeverity, boolToInt(policy.NotifyResolved),
		policy.WarningFlapThreshold, policy.GroupWaitSeconds, policy.GroupIntervalSeconds,
		policy.RepeatIntervalSeconds, boolToInt(policy.K8sEventAlertsEnabled),
		encodeJSON(receiverIDs), utcStamp(now),
	}
}

func (d *notifyPolicyDialect) ScanRow(rows *sql.Rows) (*database.NotifyPolicy, error) {
	policy := &database.NotifyPolicy{}
	var incidentEnabled, notifyResolved, k8sEnabled int
	var receiverIDs, updatedAt string
	err := rows.Scan(&incidentEnabled, &policy.MinSeverity, &notifyResolved, &policy.WarningFlapThreshold,
		&policy.GroupWaitSeconds, &policy.GroupIntervalSeconds, &policy.RepeatIntervalSeconds, &k8sEnabled,
		&receiverIDs, &updatedAt)
	if err != nil {
		return nil, err
	}
	policy.IncidentAlertsEnabled = incidentEnabled == 1
	policy.NotifyResolved = notifyResolved == 1
	policy.K8sEventAlertsEnabled = k8sEnabled == 1
	policy.DefaultReceiverIDs = decodeInt64s(receiverIDs)
	policy.UpdatedAt = parseStamp(updatedAt)
	return policy, nil
}

var _ database.NotifyPolicyDialect = (*notifyPolicyDialect)(nil)
