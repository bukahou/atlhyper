// atlhyper_master_v2/database/repo/aiops_incident.go
// AIOps 事件 Repository 实现
package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// aiopsIncidentRepo AIOps 事件 Repository 实现
type aiopsIncidentRepo struct {
	db      *sql.DB
	dialect database.AIOpsIncidentDialect
}

// newAIOpsIncidentRepo 创建 AIOps 事件 Repository
func newAIOpsIncidentRepo(db *sql.DB, dialect database.AIOpsIncidentDialect) *aiopsIncidentRepo {
	return &aiopsIncidentRepo{db: db, dialect: dialect}
}

// CreateIncident 创建事件
func (r *aiopsIncidentRepo) CreateIncident(ctx context.Context, inc *database.AIOpsIncident) error {
	query, args := r.dialect.InsertIncident(inc)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetByID 按 ID 查询事件
func (r *aiopsIncidentRepo) GetByID(ctx context.Context, id string) (*database.AIOpsIncident, error) {
	query, args := r.dialect.SelectByID(id)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	return r.dialect.ScanIncident(rows)
}

// UpdateState 更新事件状态和严重程度
func (r *aiopsIncidentRepo) UpdateState(ctx context.Context, id, state, severity string) error {
	query, args := r.dialect.UpdateState(id, state, severity)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// Resolve 解决事件（设置结束时间，计算持续时长）
func (r *aiopsIncidentRepo) Resolve(ctx context.Context, id string, resolvedAt time.Time) error {
	query, args := r.dialect.Resolve(id, resolvedAt)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// UpdateRootCause 更新根因实体
func (r *aiopsIncidentRepo) UpdateRootCause(ctx context.Context, id, rootCause string) error {
	query, args := r.dialect.UpdateRootCause(id, rootCause)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// UpdatePeakRisk 更新峰值风险分数（只增不减）
func (r *aiopsIncidentRepo) UpdatePeakRisk(ctx context.Context, id string, peakRisk float64) error {
	query, args := r.dialect.UpdatePeakRisk(id, peakRisk)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// IncrementRecurrence 递增复发次数
func (r *aiopsIncidentRepo) IncrementRecurrence(ctx context.Context, id string) error {
	query, args := r.dialect.IncrementRecurrence(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// List 按条件查询事件列表
func (r *aiopsIncidentRepo) List(ctx context.Context, opts database.AIOpsIncidentQueryOpts) ([]*database.AIOpsIncident, error) {
	query, args := buildIncidentListQuery(opts, false)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*database.AIOpsIncident
	for rows.Next() {
		inc, err := r.dialect.ScanIncident(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, inc)
	}
	return result, rows.Err()
}

// Count 按条件统计事件数量
func (r *aiopsIncidentRepo) Count(ctx context.Context, opts database.AIOpsIncidentQueryOpts) (int, error) {
	query, args := buildIncidentListQuery(opts, true)
	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// AddEntity 添加受影响实体
func (r *aiopsIncidentRepo) AddEntity(ctx context.Context, entity *database.AIOpsIncidentEntity) error {
	query, args := r.dialect.InsertEntity(entity)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetEntities 查询事件关联的实体
func (r *aiopsIncidentRepo) GetEntities(ctx context.Context, incidentID string) ([]*database.AIOpsIncidentEntity, error) {
	query, args := r.dialect.SelectEntities(incidentID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*database.AIOpsIncidentEntity
	for rows.Next() {
		e, err := r.dialect.ScanEntity(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// AddTimeline 添加时间线事件
func (r *aiopsIncidentRepo) AddTimeline(ctx context.Context, entry *database.AIOpsIncidentTimeline) error {
	query, args := r.dialect.InsertTimeline(entry)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetTimeline 查询事件时间线
func (r *aiopsIncidentRepo) GetTimeline(ctx context.Context, incidentID string) ([]*database.AIOpsIncidentTimeline, error) {
	query, args := r.dialect.SelectTimeline(incidentID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*database.AIOpsIncidentTimeline
	for rows.Next() {
		t, err := r.dialect.ScanTimeline(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// GetIncidentStats 获取事件统计信息
func (r *aiopsIncidentRepo) GetIncidentStats(ctx context.Context, clusterID string, since time.Time) (*database.AIOpsIncidentStatsRaw, error) {
	sinceStr := since.Format(time.RFC3339)

	// 总事件数
	var totalIncidents int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM aiops_incidents WHERE cluster_id = ? AND started_at >= ?`,
		clusterID, sinceStr).Scan(&totalIncidents)
	if err != nil {
		return nil, fmt.Errorf("统计总事件数失败: %w", err)
	}

	// 活跃事件数
	var activeIncidents int
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM aiops_incidents WHERE cluster_id = ? AND started_at >= ? AND state != 'stable'`,
		clusterID, sinceStr).Scan(&activeIncidents)
	if err != nil {
		return nil, fmt.Errorf("统计活跃事件数失败: %w", err)
	}

	// 平均修复时间（MTTR），仅计算已解决的事件
	var mttr sql.NullFloat64
	err = r.db.QueryRowContext(ctx,
		`SELECT AVG(duration_s) FROM aiops_incidents WHERE cluster_id = ? AND started_at >= ? AND state = 'stable' AND duration_s > 0`,
		clusterID, sinceStr).Scan(&mttr)
	if err != nil {
		return nil, fmt.Errorf("统计 MTTR 失败: %w", err)
	}

	// 复发事件数
	var recurringCount int
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM aiops_incidents WHERE cluster_id = ? AND started_at >= ? AND recurrence > 0`,
		clusterID, sinceStr).Scan(&recurringCount)
	if err != nil {
		return nil, fmt.Errorf("统计复发事件数失败: %w", err)
	}

	// 按严重程度分组统计
	bySeverity := make(map[string]int)
	sevRows, err := r.db.QueryContext(ctx,
		`SELECT severity, COUNT(*) FROM aiops_incidents WHERE cluster_id = ? AND started_at >= ? GROUP BY severity`,
		clusterID, sinceStr)
	if err != nil {
		return nil, fmt.Errorf("按严重程度统计失败: %w", err)
	}
	defer sevRows.Close()
	for sevRows.Next() {
		var sev string
		var cnt int
		if err := sevRows.Scan(&sev, &cnt); err != nil {
			return nil, err
		}
		bySeverity[sev] = cnt
	}
	if err := sevRows.Err(); err != nil {
		return nil, err
	}

	// 按状态分组统计
	byState := make(map[string]int)
	stateRows, err := r.db.QueryContext(ctx,
		`SELECT state, COUNT(*) FROM aiops_incidents WHERE cluster_id = ? AND started_at >= ? GROUP BY state`,
		clusterID, sinceStr)
	if err != nil {
		return nil, fmt.Errorf("按状态统计失败: %w", err)
	}
	defer stateRows.Close()
	for stateRows.Next() {
		var st string
		var cnt int
		if err := stateRows.Scan(&st, &cnt); err != nil {
			return nil, err
		}
		byState[st] = cnt
	}
	if err := stateRows.Err(); err != nil {
		return nil, err
	}

	mttrVal := 0.0
	if mttr.Valid {
		mttrVal = mttr.Float64
	}

	return &database.AIOpsIncidentStatsRaw{
		TotalIncidents:  totalIncidents,
		ActiveIncidents: activeIncidents,
		MTTR:            mttrVal,
		RecurringCount:  recurringCount,
		BySeverity:      bySeverity,
		ByState:         byState,
	}, nil
}

// TopRootCauses 查询频率最高的根因实体
func (r *aiopsIncidentRepo) TopRootCauses(ctx context.Context, clusterID string, since time.Time, limit int) ([]database.AIOpsRootCauseCount, error) {
	sinceStr := since.Format(time.RFC3339)
	rows, err := r.db.QueryContext(ctx,
		`SELECT root_cause, COUNT(*) as cnt FROM aiops_incidents
		WHERE cluster_id = ? AND started_at >= ? AND root_cause != '' AND root_cause IS NOT NULL
		GROUP BY root_cause ORDER BY cnt DESC LIMIT ?`,
		clusterID, sinceStr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []database.AIOpsRootCauseCount
	for rows.Next() {
		var rc database.AIOpsRootCauseCount
		if err := rows.Scan(&rc.EntityKey, &rc.Count); err != nil {
			return nil, err
		}
		result = append(result, rc)
	}
	return result, rows.Err()
}

// ListByEntity 查询与指定实体相关的事件
func (r *aiopsIncidentRepo) ListByEntity(ctx context.Context, entityKey string, since time.Time) ([]*database.AIOpsIncident, error) {
	sinceStr := since.Format(time.RFC3339)
	rows, err := r.db.QueryContext(ctx,
		`SELECT i.id, i.cluster_id, i.state, i.severity, i.root_cause, i.peak_risk, i.started_at, i.resolved_at, i.duration_s, i.recurrence, i.summary, i.created_at
		FROM aiops_incidents i
		INNER JOIN aiops_incident_entities e ON i.id = e.incident_id
		WHERE e.entity_key = ? AND i.started_at >= ?
		ORDER BY i.started_at DESC`,
		entityKey, sinceStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*database.AIOpsIncident
	for rows.Next() {
		inc, err := r.dialect.ScanIncident(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, inc)
	}
	return result, rows.Err()
}

// buildIncidentListQuery 构建事件列表查询（动态 WHERE 条件）
func buildIncidentListQuery(opts database.AIOpsIncidentQueryOpts, countOnly bool) (string, []any) {
	var conditions []string
	var args []any

	if opts.ClusterID != "" {
		conditions = append(conditions, "cluster_id = ?")
		args = append(args, opts.ClusterID)
	}
	if opts.State != "" {
		conditions = append(conditions, "state = ?")
		args = append(args, opts.State)
	}
	if opts.Severity != "" {
		conditions = append(conditions, "severity = ?")
		args = append(args, opts.Severity)
	}
	if !opts.From.IsZero() {
		conditions = append(conditions, "started_at >= ?")
		args = append(args, opts.From.Format(time.RFC3339))
	}
	if !opts.To.IsZero() {
		conditions = append(conditions, "started_at <= ?")
		args = append(args, opts.To.Format(time.RFC3339))
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	if countOnly {
		return "SELECT COUNT(*) FROM aiops_incidents" + where, args
	}

	query := `SELECT id, cluster_id, state, severity, root_cause, peak_risk, started_at, resolved_at, duration_s, recurrence, summary, created_at
		FROM aiops_incidents` + where + ` ORDER BY started_at DESC`

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}

	return query, args
}

// 确保实现了接口
var _ database.AIOpsIncidentRepository = (*aiopsIncidentRepo)(nil)
