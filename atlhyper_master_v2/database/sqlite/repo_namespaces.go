package sqlite

import (
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type repoNamespaceDialect struct{}

func (d *repoNamespaceDialect) Insert(repo, namespace string) (string, []any) {
	return `INSERT OR IGNORE INTO repo_namespaces (repo, namespace) VALUES (?, ?)`, []any{repo, namespace}
}

func (d *repoNamespaceDialect) Delete(repo, namespace string) (string, []any) {
	return `DELETE FROM repo_namespaces WHERE repo=? AND namespace=?`, []any{repo, namespace}
}

func (d *repoNamespaceDialect) SelectByRepo(repo string) (string, []any) {
	return `SELECT namespace FROM repo_namespaces WHERE repo=? ORDER BY namespace`, []any{repo}
}

func (d *repoNamespaceDialect) ScanNamespace(rows *sql.Rows) (string, error) {
	var ns string
	err := rows.Scan(&ns)
	return ns, err
}

var _ database.RepoNamespaceDialect = (*repoNamespaceDialect)(nil)
