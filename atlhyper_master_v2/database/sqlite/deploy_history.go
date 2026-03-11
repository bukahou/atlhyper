package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type deployHistoryDialect struct{}

const deployHistoryCols = "id, cluster_id, path, namespace, commit_sha, commit_message, commit_author, commit_avatar_url, pr_number, pr_title, pr_url, changed_files, compare_url, deployed_at, trigger, status, duration_ms, resource_total, resource_changed, error_message"

func (d *deployHistoryDialect) Insert(record *database.DeployHistory) (string, []any) {
	return `INSERT INTO deploy_history (cluster_id, path, namespace, commit_sha, commit_message, commit_author, commit_avatar_url, pr_number, pr_title, pr_url, changed_files, compare_url, deployed_at, trigger, status, duration_ms, resource_total, resource_changed, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{record.ClusterID, record.Path, record.Namespace, record.CommitSHA, record.CommitMessage,
			record.CommitAuthor, record.CommitAvatarURL, record.PRNumber, record.PRTitle, record.PRURL,
			record.ChangedFiles, record.CompareURL,
			record.DeployedAt.Format(time.RFC3339), record.Trigger, record.Status,
			record.DurationMs, record.ResourceTotal, record.ResourceChanged, record.ErrorMessage}
}

func (d *deployHistoryDialect) SelectByID(id int64) (string, []any) {
	return "SELECT " + deployHistoryCols + " FROM deploy_history WHERE id = ?", []any{id}
}

func (d *deployHistoryDialect) SelectWithOpts(opts database.DeployHistoryQueryOpts) (string, []any) {
	q := "SELECT " + deployHistoryCols + " FROM deploy_history WHERE 1=1"
	var args []any
	if opts.ClusterID != "" {
		q += " AND cluster_id = ?"
		args = append(args, opts.ClusterID)
	}
	if opts.Path != "" {
		q += " AND path = ?"
		args = append(args, opts.Path)
	}
	q += " ORDER BY deployed_at DESC"
	if opts.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", opts.Limit)
		if opts.Offset > 0 {
			q += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}
	return q, args
}

func (d *deployHistoryDialect) CountWithOpts(opts database.DeployHistoryQueryOpts) (string, []any) {
	q := "SELECT COUNT(*) FROM deploy_history WHERE 1=1"
	var args []any
	if opts.ClusterID != "" {
		q += " AND cluster_id = ?"
		args = append(args, opts.ClusterID)
	}
	if opts.Path != "" {
		q += " AND path = ?"
		args = append(args, opts.Path)
	}
	return q, args
}

func (d *deployHistoryDialect) SelectLatestByPath(clusterID, path string) (string, []any) {
	return "SELECT " + deployHistoryCols + " FROM deploy_history WHERE cluster_id = ? AND path = ? ORDER BY deployed_at DESC LIMIT 1",
		[]any{clusterID, path}
}

func (d *deployHistoryDialect) ScanRow(rows *sql.Rows) (*database.DeployHistory, error) {
	r := &database.DeployHistory{}
	var deployedAt string
	err := rows.Scan(&r.ID, &r.ClusterID, &r.Path, &r.Namespace, &r.CommitSHA, &r.CommitMessage,
		&r.CommitAuthor, &r.CommitAvatarURL, &r.PRNumber, &r.PRTitle, &r.PRURL,
		&r.ChangedFiles, &r.CompareURL,
		&deployedAt, &r.Trigger, &r.Status, &r.DurationMs, &r.ResourceTotal, &r.ResourceChanged, &r.ErrorMessage)
	if err != nil {
		return nil, err
	}
	r.DeployedAt, _ = time.Parse(time.RFC3339, deployedAt)
	return r, nil
}

var _ database.DeployHistoryDialect = (*deployHistoryDialect)(nil)
