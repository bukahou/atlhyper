// atlhyper_master_v2/database/sqlite/notify_rule.go
// SQLite NotifyRuleDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyRuleDialect struct{}

const notifyRuleColumns = `id, name, position, enabled, cluster_ids, sources, min_severity, receiver_ids,
	group_wait_seconds, group_interval_seconds, repeat_interval_seconds, created_at, updated_at`

func (d *notifyRuleDialect) Insert(rule *database.NotifyRule, position int, now time.Time) (string, []any) {
	stamp := utcStamp(now)
	query := `INSERT INTO notify_rules (name, position, enabled, cluster_ids, sources, min_severity, receiver_ids,
		group_wait_seconds, group_interval_seconds, repeat_interval_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return query, append([]any{rule.Name, position}, append(notifyRuleValues(rule), stamp, stamp)...)
}

func (d *notifyRuleDialect) Update(rule *database.NotifyRule, now time.Time) (string, []any) {
	query := `UPDATE notify_rules SET name = ?, enabled = ?, cluster_ids = ?, sources = ?, min_severity = ?, receiver_ids = ?,
		group_wait_seconds = ?, group_interval_seconds = ?, repeat_interval_seconds = ?, updated_at = ? WHERE id = ?`
	return query, append([]any{rule.Name}, append(notifyRuleValues(rule), utcStamp(now), rule.ID)...)
}

// notifyRuleValues 插入与更新共用的那一段列值：enabled … repeat_interval_seconds
func notifyRuleValues(rule *database.NotifyRule) []any {
	clusterIDs, sources, receiverIDs := rule.ClusterIDs, rule.Sources, rule.ReceiverIDs
	if clusterIDs == nil {
		clusterIDs = []string{}
	}
	if sources == nil {
		sources = []string{}
	}
	if receiverIDs == nil {
		receiverIDs = []int64{}
	}
	return []any{
		boolToInt(rule.Enabled), encodeJSON(clusterIDs), encodeJSON(sources), rule.MinSeverity, encodeJSON(receiverIDs),
		nullableInt(rule.GroupWaitSeconds), nullableInt(rule.GroupIntervalSeconds), nullableInt(rule.RepeatIntervalSeconds),
	}
}

func (d *notifyRuleDialect) UpdatePosition(id int64, position int) (string, []any) {
	return "UPDATE notify_rules SET position = ? WHERE id = ?", []any{position, id}
}

func (d *notifyRuleDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM notify_rules WHERE id = ?", []any{id}
}

func (d *notifyRuleDialect) SelectByID(id int64) (string, []any) {
	return "SELECT " + notifyRuleColumns + " FROM notify_rules WHERE id = ?", []any{id}
}

func (d *notifyRuleDialect) SelectAll() (string, []any) {
	return "SELECT " + notifyRuleColumns + " FROM notify_rules ORDER BY position, id", nil
}

func (d *notifyRuleDialect) SelectMaxPosition() (string, []any) {
	return "SELECT COALESCE(MAX(position), 0) FROM notify_rules", nil
}

func (d *notifyRuleDialect) ScanRow(rows *sql.Rows) (*database.NotifyRule, error) {
	rule := &database.NotifyRule{}
	var enabled int
	var clusterIDs, sources, receiverIDs, createdAt, updatedAt string
	var groupWait, groupInterval, repeatInterval sql.NullInt64
	err := rows.Scan(&rule.ID, &rule.Name, &rule.Position, &enabled, &clusterIDs, &sources, &rule.MinSeverity, &receiverIDs,
		&groupWait, &groupInterval, &repeatInterval, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	rule.Enabled = enabled == 1
	rule.ClusterIDs = decodeStrings(clusterIDs)
	rule.Sources = decodeStrings(sources)
	rule.ReceiverIDs = decodeInt64s(receiverIDs)
	rule.GroupWaitSeconds = parseNullableInt(groupWait)
	rule.GroupIntervalSeconds = parseNullableInt(groupInterval)
	rule.RepeatIntervalSeconds = parseNullableInt(repeatInterval)
	rule.CreatedAt = parseStamp(createdAt)
	rule.UpdatedAt = parseStamp(updatedAt)
	return rule, nil
}

var _ database.NotifyRuleDialect = (*notifyRuleDialect)(nil)
