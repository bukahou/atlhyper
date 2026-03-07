// atlhyper_master_v2/database/repo/ai_report.go
// AIReport Repository 实现
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiReportRepo struct {
	db      *sql.DB
	dialect database.AIReportDialect
}

func newAIReportRepo(db *sql.DB, dialect database.AIReportDialect) *aiReportRepo {
	return &aiReportRepo{db: db, dialect: dialect}
}

func (r *aiReportRepo) Create(ctx context.Context, report *database.AIReport) error {
	query, args := r.dialect.Insert(report)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	report.ID = id
	return nil
}

func (r *aiReportRepo) GetByID(ctx context.Context, id int64) (*database.AIReport, error) {
	query, args := r.dialect.SelectByID(id)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return r.dialect.ScanRow(rows)
	}
	return nil, nil
}

func (r *aiReportRepo) ListByIncident(ctx context.Context, incidentID string) ([]*database.AIReport, error) {
	query, args := r.dialect.SelectByIncident(incidentID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*database.AIReport
	for rows.Next() {
		report, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (r *aiReportRepo) ListByCluster(ctx context.Context, clusterID, role string, limit int) ([]*database.AIReport, error) {
	query, args := r.dialect.SelectByCluster(clusterID, role, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*database.AIReport
	for rows.Next() {
		report, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (r *aiReportRepo) CountByClusterAndRole(ctx context.Context, clusterID, role string, since time.Time) (int, error) {
	query, args := r.dialect.CountByClusterAndRole(clusterID, role, since)
	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *aiReportRepo) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *aiReportRepo) UpdateInvestigationSteps(ctx context.Context, id int64, steps string) error {
	query, args := r.dialect.UpdateInvestigationSteps(id, steps)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

var _ database.AIReportRepository = (*aiReportRepo)(nil)
