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
		return nil, rows.Err() // 没有行是 nil, nil；读取出错要报出来
	}
	return r.dialect.ScanIncident(rows)
}

// UpdateState 更新事件状态（不动严重度）
func (r *aiopsIncidentRepo) UpdateState(ctx context.Context, id, state string) error {
	query, args := r.dialect.UpdateState(id, state)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// RaiseSeverity 抬高峰值严重度与峰值风险（只升不降）
func (r *aiopsIncidentRepo) RaiseSeverity(ctx context.Context, id, severity string, peakRisk float64) error {
	query, args := r.dialect.RaiseSeverity(id, severity, peakRisk)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// MarkEscalated 记下首次升级时间（已有值时不覆盖）
func (r *aiopsIncidentRepo) MarkEscalated(ctx context.Context, id string, escalatedAt time.Time) error {
	query, args := r.dialect.MarkEscalated(id, escalatedAt)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// SetAnomalyDuration 记下异常持续到 endedAt 为止的时长
func (r *aiopsIncidentRepo) SetAnomalyDuration(ctx context.Context, id string, endedAt time.Time) error {
	query, args := r.dialect.SetAnomalyDuration(id, endedAt)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// Resolve 关闭事件（状态、结束时间、关闭原因；严重度保留；时长见方言注释）
func (r *aiopsIncidentRepo) Resolve(ctx context.Context, id string, resolvedAt time.Time, reason string) error {
	query, args := r.dialect.Resolve(id, resolvedAt, reason)
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
	query, args := buildIncidentListQuery(r.dialect.IncidentColumns(""), opts, false)
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
	query, args := buildIncidentListQuery("", opts, true)
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

// UpsertSignal 累计一条信号记录
func (r *aiopsIncidentRepo) UpsertSignal(ctx context.Context, signal *database.AIOpsIncidentSignal) error {
	query, args := r.dialect.UpsertSignal(signal)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetSignals 某个事件的全部信号记录（最高分从高到低）
func (r *aiopsIncidentRepo) GetSignals(ctx context.Context, incidentID string) ([]*database.AIOpsIncidentSignal, error) {
	query, args := r.dialect.SelectSignals(incidentID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []*database.AIOpsIncidentSignal{}
	for rows.Next() {
		signal, err := r.dialect.ScanSignal(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, signal)
	}
	return result, rows.Err()
}

// SaveLink 写入或更新一条关联
func (r *aiopsIncidentRepo) SaveLink(ctx context.Context, link *database.AIOpsIncidentLink) error {
	query, args := r.dialect.SaveLink(link)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// UnlinkIncident 解除受影响事件的关联
func (r *aiopsIncidentRepo) UnlinkIncident(ctx context.Context, incidentID string, at time.Time) error {
	query, args := r.dialect.UnlinkIncident(incidentID, at)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// ListActiveLinks 当前有效的关联
func (r *aiopsIncidentRepo) ListActiveLinks(ctx context.Context, clusterID string) ([]*database.AIOpsIncidentLink, error) {
	query, args := r.dialect.SelectActiveLinks(clusterID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []*database.AIOpsIncidentLink{}
	for rows.Next() {
		link, err := r.dialect.ScanLink(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, link)
	}
	return result, rows.Err()
}

// GetIncidentStats 获取事件统计信息
func (r *aiopsIncidentRepo) GetIncidentStats(ctx context.Context, clusterID string, since time.Time) (*database.AIOpsIncidentStatsRaw, error) {
	sinceStr := database.Stamp(since)

	// 总事件数
	var totalIncidents int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM aiops_incidents WHERE cluster_id = ? AND started_at >= ?`,
		clusterID, sinceStr).Scan(&totalIncidents)
	if err != nil {
		return nil, fmt.Errorf("统计总事件数失败: %w", err)
	}

	// 活跃事件数：仍在异常中的（警告 / 事件中）。恢复观察中的事件问题已经消失，不算活跃
	var activeIncidents int
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM aiops_incidents WHERE cluster_id = ? AND started_at >= ? AND state IN ('warning', 'incident')`,
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

// TopRootCauses 出事件最多的主体。
// 按主体而不是领头实体分组：Pod 重建会换名字，按 Pod 分组会把同一个工作负载拆散
func (r *aiopsIncidentRepo) TopRootCauses(ctx context.Context, clusterID string, since time.Time, limit int) ([]database.AIOpsRootCauseCount, error) {
	sinceStr := database.Stamp(since)
	rows, err := r.db.QueryContext(ctx,
		`SELECT subject, COUNT(*) as cnt FROM aiops_incidents
		WHERE cluster_id = ? AND started_at >= ? AND subject != ''
		GROUP BY subject ORDER BY cnt DESC LIMIT ?`,
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

// ListByEntity 查询某个集群里与指定实体相关的事件：主体是它，或成员实体里有它
func (r *aiopsIncidentRepo) ListByEntity(ctx context.Context, clusterID, entityKey string, since time.Time) ([]*database.AIOpsIncident, error) {
	sinceStr := database.Stamp(since)
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+r.dialect.IncidentColumns("i.")+`
		FROM aiops_incidents i
		WHERE i.cluster_id = ? AND i.started_at >= ?
			AND (i.subject = ? OR EXISTS (
				SELECT 1 FROM aiops_incident_entities e WHERE e.incident_id = i.id AND e.entity_key = ?))
		ORDER BY i.started_at DESC`,
		clusterID, sinceStr, entityKey, entityKey)
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

// buildIncidentListQuery 构建事件列表查询（动态 WHERE 条件）；columns 为方言给出的列清单，只统计数量时不用
func buildIncidentListQuery(columns string, opts database.AIOpsIncidentQueryOpts, countOnly bool) (string, []any) {
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
		args = append(args, database.Stamp(opts.From))
	}
	if !opts.To.IsZero() {
		conditions = append(conditions, "started_at <= ?")
		args = append(args, database.Stamp(opts.To))
	}
	if opts.ExcludeLinked {
		conditions = append(conditions, "id NOT IN (SELECT incident_id FROM aiops_incident_links WHERE unlinked_at IS NULL)")
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	if countOnly {
		return "SELECT COUNT(*) FROM aiops_incidents" + where, args
	}

	query := `SELECT ` + columns + ` FROM aiops_incidents` + where + ` ORDER BY started_at DESC`

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
