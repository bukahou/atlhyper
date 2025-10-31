// retriever/retriever.go
package retriever

import (
	"context"
	"fmt"
)

// RetrieveSimilarDocs —— 向量检索
// 输入：embedding（LLM 生成的向量），limit（返回条数）
// 输出：相似度最高的文档列表
func RetrieveSimilarDocs(ctx context.Context, embedding []float32, limit int) ([]Document, error) {
	db := GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized, call retriever.InitDB first")
	}

	query := `
	SELECT id, title, content
	FROM documents
	ORDER BY embedding <-> $1
	LIMIT $2;
	`

	rows, err := db.Query(ctx, query, embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.Title, &d.Content); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, nil
}
