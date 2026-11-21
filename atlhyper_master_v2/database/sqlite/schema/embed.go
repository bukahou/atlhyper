// Package schema SQLite 的表结构迁移（goose）。
//
// 文件名 NNNNN_说明.sql，版本号只增不改；已发布的迁移 ⛔ 不再修改，改表一律新增一个文件。
package schema

import "embed"

// FS 全部迁移文件
//
//go:embed *.sql
var FS embed.FS
