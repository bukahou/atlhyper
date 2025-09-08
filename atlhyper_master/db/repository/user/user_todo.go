// atlhyper_master/db/repository/user/user_todo.go
package user

import (
	"AtlHyper/atlhyper_master/db/utils"
	"database/sql"
	"time"
)

type Todo struct {
	ID        int64   `json:"id"`
	Username  string  `json:"username"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	IsDone    int     `json:"is_done"`
	DueDate   *string `json:"due_date"`  
	Priority  int     `json:"priority"`
	Category  string  `json:"category"`
	Deleted   int     `json:"deleted"`
}

// GetTodosByUser 根据用户名获取代办事项
func GetTodosByUsername(username string) ([]Todo, error) {
	rows, err := utils.DB.Query(`SELECT id, username, title, content, created_at, updated_at, is_done, due_date, priority, category, deleted FROM todos WHERE username = ? AND deleted = 0`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		err := rows.Scan(&t.ID, &t.Username, &t.Title, &t.Content, &t.CreatedAt, &t.UpdatedAt, &t.IsDone, &t.DueDate, &t.Priority, &t.Category, &t.Deleted)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, nil
}

// GetTodoByID 根据 ID 获取单条代办（仅未删除）
func GetTodoByID(id int64) (*Todo, error) {
	row := utils.DB.QueryRow(`
		SELECT id, username, title, content, created_at, updated_at, is_done, due_date, priority, category, deleted
		FROM todos
		WHERE id = ? AND deleted = 0
		LIMIT 1
	`, id)

	var t Todo
	if err := row.Scan(
		&t.ID,
		&t.Username,
		&t.Title,
		&t.Content,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.IsDone,
		&t.DueDate,   // *string，NULL 会自动变为 nil
		&t.Priority,
		&t.Category,
		&t.Deleted,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 没找到
		}
		return nil, err
	}
	return &t, nil
}



// GetAllTodos 获取所有代办事项
func GetAllTodos() ([]Todo, error) {
	rows, err := utils.DB.Query(`SELECT id, username, title, content, created_at, updated_at, is_done, due_date, priority, category, deleted FROM todos WHERE deleted = 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		err := rows.Scan(&t.ID, &t.Username, &t.Title, &t.Content, &t.CreatedAt, &t.UpdatedAt, &t.IsDone, &t.DueDate, &t.Priority, &t.Category, &t.Deleted)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, nil
}


// AddTodo 新增代办事项
func AddTodo(todo Todo) error {
	_, err := utils.DB.Exec(`
		INSERT INTO todos (username, title, content, created_at, is_done, due_date, priority, category, deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		todo.Username,
		todo.Title,
		todo.Content,
		time.Now().Format("2006-01-02 15:04:05"), // created_at
		todo.IsDone,
		todo.DueDate,
		todo.Priority,
		todo.Category,
		0, // 默认 not deleted
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
		time.Now().Format("2006-01-02 15:04:05"), // updated_at
		todo.IsDone,
		todo.DueDate,
		todo.Priority,
		todo.Category,
		todo.Deleted,
		todo.ID,
	)
	return err
}
