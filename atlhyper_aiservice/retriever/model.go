// atlhyper_aiservice/retriever/model.go
package retriever

import "time"

// Document —— 向量数据库中的文档结构体
// 对应 PostgreSQL 中的 knowledge_entries 或 documents 表
type Document struct {
	ID        int       `db:"id"`
	Title     string    `db:"title"`
	Content   string    `db:"content"`
	Embedding []float32 `db:"embedding"` // pgvector 向量字段
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
