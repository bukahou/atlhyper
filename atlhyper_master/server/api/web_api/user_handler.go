// atlhyper_master/server/api/web_api/user.go
package web_api

import (
	repo "AtlHyper/atlhyper_master/db/repository/user"
	uiuser "AtlHyper/atlhyper_master/interfaces/ui_interfaces/user"
	"AtlHyper/atlhyper_master/server/api/response"
	"time"

	"github.com/gin-gonic/gin"
)

// 获取指定用户代办事项
func GetUserTodosHandler(c *gin.Context) {

	var req struct {
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取用户代办失败: 参数错误")
		return
	}
	username := req.Username
	if username == "" {
		response.Error(c, "username 不能为空")
		return
	}

	todos, err := uiuser.GetUserTodos(username)
	if err != nil {
		response.ErrorCode(c, 50000, "获取用户代办失败: "+err.Error())
		return
	}
	response.Success(c, "获取成功", gin.H{
		"username": username,
		"items":    todos,
	})
}

// 获取全部代办事项
func GetAllTodosHandler(c *gin.Context) {
	todos, err := uiuser.GetAllTodos()
	if err != nil {
		response.ErrorCode(c, 50000, "获取全部代办失败: "+err.Error())
		return
	}
	response.Success(c, "获取成功", gin.H{
		"total": len(todos),
		"items": todos,
	})
}

// 新增代办
func CreateTodoHandler(c *gin.Context) {
	var in repo.Todo
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, "请求体解析失败: "+err.Error())
		return
	}
	// 业务校验在 interfaces 层也有，这里再兜底一次
	if in.Username == "" {
		response.Error(c, "username 不能为空")
		return
	}
	if in.Title == "" {
		response.Error(c, "title 不能为空")
		return
	}

	if err := uiuser.CreateTodo(in); err != nil {
		// interfaces 层会返回 ErrInvalidInput 等
		response.ErrorCode(c, 50000, "新增代办失败: "+err.Error())
		return
	}
	response.SuccessMsg(c, "新增成功")
}


// 更新代办（JSON-only，支持只更新 title/content/is_done/priority/deleted）
func UpdateTodoHandler(c *gin.Context) {
	type UpdateReq struct {
		ID       int64   `json:"id" binding:"required"`
		Title    *string `json:"title,omitempty"`
		Content  *string `json:"content,omitempty"`
		IsDone   *int    `json:"is_done,omitempty"`   // 0/1
		Priority *int    `json:"priority,omitempty"`  // 1/2/3
		Deleted  *int    `json:"deleted,omitempty"`   // 0/1
	}

	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取参数失败: "+err.Error())
		return
	}
	if req.ID <= 0 {
		response.Error(c, "id 不能为空")
		return
	}

	// 1) 读取原记录，确保未传字段不被清空
	old, err := uiuser.GetUserTodoByID(req.ID)
	if err != nil {
		response.ErrorCode(c, 50000, "读取原始数据失败: "+err.Error())
		return
	}
	if old == nil {
		response.Error(c, "记录不存在")
		return
	}

	// 2) 合并：仅覆盖这 5 个允许的字段
	in := *old
	if req.Title != nil {
		in.Title = *req.Title
	}
	if req.Content != nil {
		in.Content = *req.Content
	}
	if req.IsDone != nil {
		if *req.IsDone != 0 && *req.IsDone != 1 {
			response.Error(c, "is_done 只能为 0 或 1")
			return
		}
		in.IsDone = *req.IsDone
	}
	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 3 {
			response.Error(c, "priority 只能为 1/2/3")
			return
		}
		in.Priority = *req.Priority
	}
	if req.Deleted != nil {
		if *req.Deleted != 0 && *req.Deleted != 1 {
			response.Error(c, "deleted 只能为 0 或 1")
			return
		}
		in.Deleted = *req.Deleted
	}
	in.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	// 3) 落库（interfaces 会调用 repository 的 UpdateTodo）
	if err := uiuser.UpdateTodo(in); err != nil {
		response.ErrorCode(c, 50000, "更新代办失败: "+err.Error())
		return
	}
	response.SuccessMsg(c, "更新成功")
}

