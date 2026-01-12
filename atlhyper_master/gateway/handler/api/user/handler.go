// gateway/handler/api/user/handler.go
// 用户管理处理器
package user

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	userSvc "AtlHyper/atlhyper_master/service/db/user"

	"github.com/gin-gonic/gin"
)

// HandleLogin 用户登录
// POST /uiapi/user/login
func HandleLogin(c *gin.Context) {
	var req userSvc.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数错误")
		return
	}

	result, err := userSvc.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c, "登录成功", gin.H{
		"token":       result.Token,
		"user":        result.User,
		"cluster_ids": result.ClusterIDs,
	})
}

// HandleListAllUsers 获取所有用户列表
// GET /uiapi/user/list
func HandleListAllUsers(c *gin.Context) {
	users, err := userSvc.GetAllUsers(c.Request.Context())
	if err != nil {
		response.Error(c, "获取用户列表失败: "+err.Error())
		return
	}

	response.Success(c, "获取用户列表成功", users)
}

// HandleRegisterUser 注册新用户
// POST /uiapi/user/register
func HandleRegisterUser(c *gin.Context) {
	var req userSvc.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效: "+err.Error())
		return
	}

	user, err := userSvc.Register(c.Request.Context(), req)
	if err != nil {
		response.ErrorCode(c, 50000, "注册失败: "+err.Error())
		return
	}

	response.Success(c, "注册成功", user)
}

// HandleUpdateUserRole 更新用户角色
// POST /uiapi/user/update-role
func HandleUpdateUserRole(c *gin.Context) {
	var req userSvc.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效: "+err.Error())
		return
	}

	if err := userSvc.UpdateRole(c.Request.Context(), req); err != nil {
		response.ErrorCode(c, 50000, "更新角色失败: "+err.Error())
		return
	}

	response.SuccessMsg(c, "角色更新成功")
}

// HandleDeleteUser 删除用户
// POST /uiapi/user/delete
func HandleDeleteUser(c *gin.Context) {
	var req userSvc.DeleteUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效: "+err.Error())
		return
	}

	// 从 JWT 中获取当前操作者 ID
	operatorIDVal, exists := c.Get("user_id")
	if !exists {
		response.ErrorCode(c, 40100, "无法获取当前用户信息")
		return
	}

	// JWT 解析出的数字是 float64
	operatorID := int(operatorIDVal.(float64))

	if err := userSvc.Delete(c.Request.Context(), req.ID, operatorID); err != nil {
		response.ErrorCode(c, 50000, "删除失败: "+err.Error())
		return
	}

	response.SuccessMsg(c, "用户删除成功")
}
