package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type gitHubInstallDialect struct{}

func (d *gitHubInstallDialect) Upsert(inst *database.GitHubInstallation) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return `INSERT INTO github_installations (id, installation_id, account_login, created_at)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET installation_id = excluded.installation_id, account_login = excluded.account_login`,
		[]any{inst.InstallationID, inst.AccountLogin, now}
}

func (d *gitHubInstallDialect) Select() (string, []any) {
	return "SELECT id, installation_id, account_login, created_at FROM github_installations WHERE id = 1", nil
}

func (d *gitHubInstallDialect) Delete() (string, []any) {
	return "DELETE FROM github_installations WHERE id = 1", nil
}

func (d *gitHubInstallDialect) ScanRow(rows *sql.Rows) (*database.GitHubInstallation, error) {
	inst := &database.GitHubInstallation{}
	var createdAt string
	err := rows.Scan(&inst.ID, &inst.InstallationID, &inst.AccountLogin, &createdAt)
	if err != nil {
		return nil, err
	}
	inst.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return inst, nil
}

var _ database.GitHubInstallDialect = (*gitHubInstallDialect)(nil)
