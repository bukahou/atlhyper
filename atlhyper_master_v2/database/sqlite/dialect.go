// atlhyper_master_v2/database/sqlite/dialect.go
// SQLite Dialect 实现
package sqlite

import (
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

// Dialect SQLite 方言
type Dialect struct {
	audit           *auditDialect
	user            *userDialect
	event           *eventDialect
	notify          *notifyDialect
	cluster         *clusterDialect
	command         *commandDialect
	settings        *settingsDialect
	aiConversation  *aiConversationDialect
	aiMessage       *aiMessageDialect
	aiProvider      *aiProviderDialect
	aiActiveConfig  *aiActiveConfigDialect
	aiProviderModel *aiProviderModelDialect
	slo             *sloDialect
	nodeMetrics     *nodeMetricsDialect
}

// NewDialect 创建 SQLite 方言
func NewDialect() *Dialect {
	return &Dialect{
		audit:           &auditDialect{},
		user:            &userDialect{},
		event:           &eventDialect{},
		notify:          &notifyDialect{},
		cluster:         &clusterDialect{},
		command:         &commandDialect{},
		settings:        &settingsDialect{},
		aiConversation:  &aiConversationDialect{},
		aiMessage:       &aiMessageDialect{},
		aiProvider:      &aiProviderDialect{},
		aiActiveConfig:  &aiActiveConfigDialect{},
		aiProviderModel: &aiProviderModelDialect{},
		slo:             &sloDialect{},
		nodeMetrics:     &nodeMetricsDialect{},
	}
}

func (d *Dialect) Audit() database.AuditDialect                   { return d.audit }
func (d *Dialect) User() database.UserDialect                     { return d.user }
func (d *Dialect) Event() database.EventDialect                   { return d.event }
func (d *Dialect) Notify() database.NotifyDialect                 { return d.notify }
func (d *Dialect) Cluster() database.ClusterDialect               { return d.cluster }
func (d *Dialect) Command() database.CommandDialect               { return d.command }
func (d *Dialect) Settings() database.SettingsDialect             { return d.settings }
func (d *Dialect) AIConversation() database.AIConversationDialect { return d.aiConversation }
func (d *Dialect) AIMessage() database.AIMessageDialect           { return d.aiMessage }
func (d *Dialect) AIProvider() database.AIProviderDialect         { return d.aiProvider }
func (d *Dialect) AIActiveConfig() database.AIActiveConfigDialect { return d.aiActiveConfig }
func (d *Dialect) AIProviderModel() database.AIProviderModelDialect { return d.aiProviderModel }
func (d *Dialect) SLO() database.SLODialect                         { return d.slo }
func (d *Dialect) NodeMetrics() database.NodeMetricsDialect         { return d.nodeMetrics }

func (d *Dialect) Migrate(db *sql.DB) error {
	return migrate(db)
}

// 确保实现了接口
var _ database.Dialect = (*Dialect)(nil)
