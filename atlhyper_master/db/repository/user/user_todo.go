// atlhyper_master/db/repository/user/user_todo.go
package user

import (
	"AtlHyper/atlhyper_master/db/utils"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ====== 数据模型 ======
type Todo struct {
	ID        int64   `json:"id"`
	Username  string  `json:"username"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	IsDone    int     `json:"is_done"`
	DueDate   *string `json:"due_date"` // NULL -> nil
	Priority  int     `json:"priority"`
	Category  string  `json:"category"`
	Deleted   int     `json:"deleted"`
}

// ====== 内部常量/工具 ======

// 统一列清单（注意 updated_at 用 COALESCE 防 NULL）
const selectColumns = `
	id, username, title, content,
	created_at,
	COALESCE(updated_at, created_at) AS updated_at,
	is_done, due_date, priority, category, deleted
`

// 默认排序：新建时间倒序
const defaultOrderBy = "ORDER BY created_at DESC"

// 新增：优先级 + 截止时间 的综合排序
// 说明：priority 1/2/3（1最高），有 due_date 的排在前面并按最近在前；
//      无 due_date 的排在后面并按 created_at 新到旧。
// 注意：due_date 为 "YYYY-MM-DD" 文本，按字典序即时间序；created_at 为 "YYYY-MM-DD HH:MM:SS"
const orderByPriorityThenDue = `
ORDER BY
  priority ASC,
  CASE WHEN due_date IS NULL OR due_date = '' THEN 1 ELSE 0 END ASC,
  CASE WHEN due_date IS NULL OR due_date = '' THEN NULL ELSE due_date END ASC,
  CASE WHEN due_date IS NULL OR due_date = '' THEN created_at ELSE NULL END DESC
`

// 时间统一出口：如需改为 UTC/ISO8601，只改这里
func nowStr() string {
	// 建议：持久化 UTC（前端展示再转时区）
	// return time.Now().UTC().Format(time.RFC3339) // 如改为 ISO8601
	return time.Now().Format("2006-01-02 15:04:05")
}

// 通用扫描器（rows/row 都可以）
func scanTodo(scanner interface {
	Scan(dest ...any) error
}) (Todo, error) {
	var t Todo
	err := scanner.Scan(
		&t.ID,
		&t.Username,
		&t.Title,
		&t.Content,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.IsDone,
		&t.DueDate,
		&t.Priority,
		&t.Category,
		&t.Deleted,
	)
	return t, err
}

// ====== 基础查询（保留原签名，内部做稳健处理）======

// GetTodosByUsername 根据用户名获取代办事项（未删除，按创建时间倒序）
// func GetTodosByUsername(username string) ([]Todo, error) {
// 	rows, err := utils.DB.Query(
// 		fmt.Sprintf(`
// 			SELECT %s
// 			FROM todos
// 			WHERE username = ? AND deleted = 0
// 			%s
// 		`, selectColumns, defaultOrderBy),
// 		username,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var todos []Todo
// 	for rows.Next() {
// 		t, err := scanTodo(rows)
// 		if err != nil {
// 			return nil, err
// 		}
// 		todos = append(todos, t)
// 	}
// 	return todos, rows.Err()
// }

func GetTodosByUsername(username string) ([]Todo, error) {
	rows, err := utils.DB.Query(
		fmt.Sprintf(`
			SELECT %s
			FROM todos
			WHERE username = ? AND deleted = 0
			%s
		`, selectColumns, orderByPriorityThenDue),
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}


// GetTodoByID 根据 ID 获取单条代办（仅未删除）
func GetTodoByID(id int64) (*Todo, error) {
	row := utils.DB.QueryRow(
		fmt.Sprintf(`
			SELECT %s
			FROM todos
			WHERE id = ? AND deleted = 0
			LIMIT 1
		`, selectColumns),
		id,
	)

	t, err := scanTodo(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// GetAllTodos 获取所有代办事项（未删除，按创建时间倒序）
func GetAllTodos() ([]Todo, error) {
	rows, err := utils.DB.Query(
		fmt.Sprintf(`
			SELECT %s
			FROM todos
			WHERE deleted = 0
			%s
		`, selectColumns, defaultOrderBy),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

// AddTodo 新增代办事项（同步写入 created_at & updated_at）
func AddTodo(todo Todo) error {
	now := nowStr()
	_, err := utils.DB.Exec(`
		INSERT INTO todos (
			username, title, content,
			created_at, updated_at,
			is_done, due_date, priority, category, deleted
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		todo.Username,
		todo.Title,
		todo.Content,
		now, // created_at
		now, // updated_at（与 created_at 同步）
		todo.IsDone,
		todo.DueDate,
		todo.Priority,
		todo.Category,
		0, // not deleted
	)
	return err
}

// UpdateTodo 更新代办事项（根据 ID）
func UpdateTodo(todo Todo) error {
	_, err := utils.DB.Exec(`
		UPDATE todos 
		SET title = ?, content = ?, updated_at = ?, is_done = ?, due_date = ?, priority = ?, category = ?, deleted = ?
		WHERE id = ?`,
		todo.Title,
		todo.Content,
		nowStr(), // 统一走工具方法
		todo.IsDone,
		todo.DueDate,
		todo.Priority,
		todo.Category,
		todo.Deleted,
		todo.ID,
	)
	return err
}

// ====== 进阶能力（不破坏兼容，供上层自选使用）======

// ListTodosFiltered 支持按条件过滤 + 分页
//  - username/isDone/priority/category 均为可选
//  - limit/offset 为分页控制（limit<=0 时不分页）
//  - 返回（items, total, err）方便前端分页
func ListTodosFiltered(
	username *string,
	isDone *int,
	priority *int,
	category *string,
	limit, offset int,
) ([]Todo, int, error) {
	// 动态 WHERE
	conds := []string{"deleted = 0"}
	args := []any{}

	if username != nil && *username != "" {
		conds = append(conds, "username = ?")
		args = append(args, *username)
	}
	if isDone != nil && (*isDone == 0 || *isDone == 1) {
		conds = append(conds, "is_done = ?")
		args = append(args, *isDone)
	}
	if priority != nil && *priority >= 1 && *priority <= 3 {
		conds = append(conds, "priority = ?")
		args = append(args, *priority)
	}
	if category != nil && *category != "" {
		conds = append(conds, "category = ?")
		args = append(args, *category)
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	// 统计总数
	var total int
	countSQL := `SELECT COUNT(1) FROM todos ` + where
	if err := utils.DB.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 列表查询
	listSQL := fmt.Sprintf(`
		SELECT %s
		FROM todos
		%s
		%s
	`, selectColumns, where, defaultOrderBy)

	if limit > 0 {
		listSQL += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := utils.DB.Query(listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []Todo
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, t)
	}
	return items, total, rows.Err()
}

// SoftDeleteTodo 软删除（deleted=1 + 更新 updated_at）
func SoftDeleteTodo(id int64) error {
	_, err := utils.DB.Exec(`
		UPDATE todos
		SET deleted = 1, updated_at = ?
		WHERE id = ? AND deleted = 0
	`, nowStr(), id)
	return err
}
