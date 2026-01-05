// atlhyper_master/server/api/web_api/user.go
package web_api

import (
	repo "AtlHyper/atlhyper_master/db/repository/user"
	uiuser "AtlHyper/atlhyper_master/service/user"
	"AtlHyper/atlhyper_master/server/api/response"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ================== 工具：错误类型识别 ==================
func isInvalidInput(err error) bool {
	_, ok := err.(uiuser.ErrInvalidInput)
	return ok
}

func isNotFound(err error) bool {
	return errors.Is(err, uiuser.ErrNotFound)
}

// ================== 1) 获取指定用户的代办事项 ==================
// 建议路由：POST /user/todos/by-username
// Body: { "username": "xxx" }
// 如已接入 JWT，可优先从 c.Get("username") 取，防止越权
func GetUserTodosHandler(c *gin.Context) {
	// 优先从鉴权上下文拿用户名（如果你有中间件设置）
	if v, ok := c.Get("username"); ok {
		if uname, _ := v.(string); uname != "" {
			todos, err := uiuser.GetUserTodos(uname)
			if err != nil {
				if isInvalidInput(err) {
					response.Error(c, err.Error())
					return
				}
				response.ErrorCode(c, 50000, "获取用户代办失败: "+err.Error())
				return
			}
			response.Success(c, "获取成功", gin.H{
				"username": uname,
				"items":    todos,
			})
			return
		}
	}

	// 兜底：从 Body 读取
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" {
		response.Error(c, "获取用户代办失败: 参数错误")
		return
	}

	todos, err := uiuser.GetUserTodos(req.Username)
	if err != nil {
		if isInvalidInput(err) {
			response.Error(c, err.Error())
			return
		}
		response.ErrorCode(c, 50000, "获取用户代办失败: "+err.Error())
		return
	}
	response.Success(c, "获取成功", gin.H{
		"username": req.Username,
		"items":    todos,
	})
}

// ================== 2) 获取全部代办事项（支持分页/过滤） ==================
// 建议路由：GET /user/todos?username=&is_done=&priority=&category=&limit=&offset=
// 注意：若对权限有要求，这里应限制为 admin
func GetAllTodosHandler(c *gin.Context) {
	// 可选解析分页/过滤
	var (
		username = c.Query("username")
		category = c.Query("category")
		limit    = parseIntDefault(c.Query("limit"), 0)
		offset   = parseIntDefault(c.Query("offset"), 0)
	)

	var isDonePtr *int
	if s := c.Query("is_done"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && (v == 0 || v == 1) {
			isDonePtr = &v
		} else {
			response.Error(c, "is_done 只能为 0 或 1")
			return
		}
	}

	var prioPtr *int
	if s := c.Query("priority"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 1 && v <= 3 {
			prioPtr = &v
		} else {
			response.Error(c, "priority 只能为 1/2/3")
			return
		}
	}

	var unamePtr *string
	if username != "" {
		unamePtr = &username
	}
	var categoryPtr *string
	if category != "" {
		categoryPtr = &category
	}

	// 如果没传任何过滤且不分页，沿用你原来的“全量列表”
	if unamePtr == nil && isDonePtr == nil && prioPtr == nil && categoryPtr == nil && limit <= 0 {
		todos, err := uiuser.GetAllTodos()
		if err != nil {
			response.ErrorCode(c, 50000, "获取全部代办失败: "+err.Error())
			return
		}
		response.Success(c, "获取成功", gin.H{
			"total": len(todos),
			"items": todos,
		})
		return
	}

	// 否则走分页/过滤通道
	res, err := uiuser.ListTodosFiltered(uiuser.ListParams{
		Username: unamePtr,
		IsDone:   isDonePtr,
		Priority: prioPtr,
		Category: categoryPtr,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		response.ErrorCode(c, 50000, "查询失败: "+err.Error())
		return
	}
	response.Success(c, "获取成功", gin.H{
		"total": res.Total,
		"items": res.Items,
	})
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}

// ================== 3) 新增代办 ==================
// 建议路由：POST /user/todo/create
func CreateTodoHandler(c *gin.Context) {
	var in repo.Todo
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, "请求体解析失败: "+err.Error())
		return
	}

	// 若有鉴权上下文，可强制覆盖 username，防止越权创建到他人名下
	if v, ok := c.Get("username"); ok {
		if uname, _ := v.(string); uname != "" {
			in.Username = uname
		}
	}

	if err := uiuser.CreateTodo(in); err != nil {
		if isInvalidInput(err) {
			response.Error(c, err.Error())
			return
		}
		response.ErrorCode(c, 50000, "新增代办失败: "+err.Error())
		return
	}
	response.SuccessMsg(c, "新增成功")
}

// ================== 4) 更新代办 ==================
// 建议路由：POST /user/todo/update
// JSON-only；允许更新 title/content/is_done/priority/deleted/due_date/category
func UpdateTodoHandler(c *gin.Context) {
	type UpdateReq struct {
		ID       int64   `json:"id" binding:"required"`
		Title    *string `json:"title,omitempty"`
		Content  *string `json:"content,omitempty"`
		IsDone   *int    `json:"is_done,omitempty"`   // 0/1
		Priority *int    `json:"priority,omitempty"`  // 1/2/3
		Deleted  *int    `json:"deleted,omitempty"`   // 0/1
		DueDate  *string `json:"due_date,omitempty"`  // "" 表示清空
		Category *string `json:"category,omitempty"`
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

	// 读原记录（interfaces 已做 not found 区分）
	old, err := uiuser.GetUserTodoByID(req.ID)
	if err != nil {
		if isNotFound(err) {
			response.Error(c, "记录不存在")
			return
		}
		if isInvalidInput(err) {
			response.Error(c, err.Error())
			return
		}
		response.ErrorCode(c, 50000, "读取原始数据失败: "+err.Error())
		return
	}

	in := *old
	if req.Title != nil {
		in.Title = *req.Title
	}
	if req.Content != nil {
		in.Content = *req.Content
	}
	if req.IsDone != nil {
		in.IsDone = *req.IsDone
	}
	if req.Priority != nil {
		in.Priority = *req.Priority
	}
	if req.Deleted != nil {
		in.Deleted = *req.Deleted
	}
	// 新增：due_date/category
	if req.DueDate != nil {
		in.DueDate = req.DueDate // interfaces 内部会校验格式与是否置空
	}
	if req.Category != nil {
		in.Category = *req.Category
	}

	// 时间交给 repo，避免分层穿透
	in.UpdatedAt = ""

	if err := uiuser.UpdateTodo(in); err != nil {
		if isInvalidInput(err) {
			response.Error(c, err.Error())
			return
		}
		if isNotFound(err) {
			response.Error(c, "记录不存在")
			return
		}
		response.ErrorCode(c, 50000, "更新代办失败: "+err.Error())
		return
	}
	response.SuccessMsg(c, "更新成功")
}

// ================== 5) 软删除（可选） ==================
// 建议路由：POST /user/todo/delete
// Body: { "id": 123 }
func SoftDeleteTodoHandler(c *gin.Context) {
	var req struct {
		ID int64 `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取参数失败: "+err.Error())
		return
	}
	if req.ID <= 0 {
		response.Error(c, "id 不能为空")
		return
	}

	if err := uiuser.SoftDeleteTodo(req.ID); err != nil {
		if isInvalidInput(err) {
			response.Error(c, err.Error())
			return
		}
		if isNotFound(err) {
			response.Error(c, "记录不存在")
			return
		}
		response.ErrorCode(c, 50000, "删除失败: "+err.Error())
		return
	}
	response.SuccessMsg(c, "删除成功")
}
