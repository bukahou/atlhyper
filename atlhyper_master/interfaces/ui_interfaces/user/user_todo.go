// atlhyper_master/interfaces/ui_interfaces/user/user_todo.go
package user

import (
	"AtlHyper/atlhyper_master/db/repository/user"
)

// GetUserTodos 获取指定用户的代办事项
func GetUserTodos(username string) ([]user.Todo, error) {
	return user.GetTodosByUsername(username)
}

// GetUserTodoByID 获取指定 ID 的代办事项
func GetUserTodoByID(id int64) (*user.Todo, error) {
	return user.GetTodoByID(id)
}

// GetAllTodos 获取所有代办事项
func GetAllTodos() ([]user.Todo, error) {
	return user.GetAllTodos()
}

// CreateTodo 新增代办事项
func CreateTodo(todo user.Todo) error {
	// 可以加业务逻辑：比如必填校验、默认值设置
	if todo.Username == "" {
		return ErrInvalidInput("username 不能为空")
	}
	if todo.Title == "" {
		return ErrInvalidInput("title 不能为空")
	}
	return user.AddTodo(todo)
}

// UpdateTodo 更新代办事项
func UpdateTodo(todo user.Todo) error {
	// 可以加业务逻辑：比如 ID 必须存在
	if todo.ID == 0 {
		return ErrInvalidInput("id 必须指定")
	}
	return user.UpdateTodo(todo)
}

// ====== 错误类型（方便扩展） ======
type ErrInvalidInput string

func (e ErrInvalidInput) Error() string {
	return string(e)
}
