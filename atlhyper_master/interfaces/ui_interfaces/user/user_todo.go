// atlhyper_master/interfaces/ui_interfaces/user/user_todo.go
package user

import (
	repo "AtlHyper/atlhyper_master/db/repository/user"
	"errors"
	"time"
)

// ================== 对外数据面（直接复用 repo.Todo） ==================
// 按你的分层规范，interfaces 层不必重新定义 DTO；如果将来要解耦，可以在这里
// 定义 UI 用 DTO 并做 repo <-> dto 的映射。

// ================== 公共错误类型 ==================
type ErrInvalidInput string
func (e ErrInvalidInput) Error() string { return string(e) }

var ErrNotFound = errors.New("record not found")

// ================== 基础读取接口（保持原签名） ==================

func GetUserTodos(username string) ([]repo.Todo, error) {
	if username == "" {
		return nil, ErrInvalidInput("username 不能为空")
	}
	return repo.GetTodosByUsername(username)
}

func GetUserTodoByID(id int64) (*repo.Todo, error) {
	if id <= 0 {
		return nil, ErrInvalidInput("id 不能为空")
	}
	t, err := repo.GetTodoByID(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}
	return t, nil
}

func GetAllTodos() ([]repo.Todo, error) {
	return repo.GetAllTodos()
}

// ================== 写入/更新（增强校验 + 默认值） ==================

// CreateTodo 新增代办事项（增强校验 + 默认值）
func CreateTodo(in repo.Todo) error {
	if in.Username == "" {
		return ErrInvalidInput("username 不能为空")
	}
	if in.Title == "" {
		return ErrInvalidInput("title 不能为空")
	}
	// is_done 只能 0/1（空值按 0 处理）
	if in.IsDone != 0 && in.IsDone != 1 {
		return ErrInvalidInput("is_done 只能为 0 或 1")
	}
	// priority 默认 2，且只能 1/2/3
	if in.Priority == 0 {
		in.Priority = 2
	}
	if in.Priority < 1 || in.Priority > 3 {
		return ErrInvalidInput("priority 只能为 1/2/3")
	}
	// due_date 格式（可选；支持 YYYY-MM-DD；如需到秒可替换为 "2006-01-02 15:04:05"）
	if in.DueDate != nil && *in.DueDate != "" {
		if _, err := time.Parse("2006-01-02", *in.DueDate); err != nil {
			return ErrInvalidInput("due_date 格式需为 YYYY-MM-DD")
		}
	}
	// category（可选，简单长度限制；如需白名单可在此添加）
	if len(in.Category) > 128 {
		return ErrInvalidInput("category 过长（<=128）")
	}

	// 由 repository 负责 created_at / updated_at 的写入；这里不触碰时间字段
	in.CreatedAt = ""
	in.UpdatedAt = ""

	return repo.AddTodo(in)
}

// UpdateTodo 更新代办事项（仅允许修改 title/content/is_done/priority/deleted/due_date/category）
func UpdateTodo(in repo.Todo) error {
	if in.ID == 0 {
		return ErrInvalidInput("id 必须指定")
	}

	// 读取原数据，保证“仅更新允许的字段”
	old, err := repo.GetTodoByID(in.ID)
	if err != nil {
		return err
	}
	if old == nil {
		return ErrNotFound
	}

	// 允许字段覆盖
	if in.Title != "" {
		old.Title = in.Title
	}
	if in.Content != "" {
		old.Content = in.Content
	}
	// is_done 校验（如果调用者传了非法值就报错；如果根本不想改，则保持旧值：调用者应传 old.IsDone）
	if in.IsDone != old.IsDone {
		if in.IsDone != 0 && in.IsDone != 1 {
			return ErrInvalidInput("is_done 只能为 0 或 1")
		}
		old.IsDone = in.IsDone
	}
	// priority 校验（同上）
	if in.Priority != 0 && in.Priority != old.Priority {
		if in.Priority < 1 || in.Priority > 3 {
			return ErrInvalidInput("priority 只能为 1/2/3")
		}
		old.Priority = in.Priority
	}
	// deleted 只能 0/1（如果想做软删除，推荐调用 SoftDeleteTodo）
	if in.Deleted != old.Deleted {
		if in.Deleted != 0 && in.Deleted != 1 {
			return ErrInvalidInput("deleted 只能为 0 或 1")
		}
		old.Deleted = in.Deleted
	}
	// due_date 支持置空（传入 "" → 置为 nil）
	if in.DueDate != nil {
		if *in.DueDate == "" {
			old.DueDate = nil
		} else {
			if _, err := time.Parse("2006-01-02", *in.DueDate); err != nil {
				return ErrInvalidInput("due_date 格式需为 YYYY-MM-DD")
			}
			old.DueDate = in.DueDate
		}
	}
	// category 替换（可附加白名单/长度校验）
	if in.Category != "" && in.Category != old.Category {
		if len(in.Category) > 128 {
			return ErrInvalidInput("category 过长（<=128）")
		}
		old.Category = in.Category
	}

	// 由 repository 层统一写 updated_at
	old.UpdatedAt = ""
	return repo.UpdateTodo(*old)
}

// ================== 进阶能力（可选暴露给 handler 用） ==================

// ListParams 供上层（handler）做分页/过滤查询
type ListParams struct {
	Username *string
	IsDone   *int   // 0 or 1
	Priority *int   // 1/2/3
	Category *string
	Limit    int
	Offset   int
}

type ListResult struct {
	Items []repo.Todo `json:"items"`
	Total int         `json:"total"`
}

func ListTodosFiltered(p ListParams) (ListResult, error) {
	items, total, err := repo.ListTodosFiltered(
		p.Username,
		p.IsDone,
		p.Priority,
		p.Category,
		p.Limit,
		p.Offset,
	)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Total: total}, nil
}

// 软删除（置 deleted=1）
func SoftDeleteTodo(id int64) error {
	if id <= 0 {
		return ErrInvalidInput("id 不能为空")
	}
	// 若不存在，返回 ErrNotFound 更友好
	t, err := repo.GetTodoByID(id)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrNotFound
	}
	return repo.SoftDeleteTodo(id)
}
