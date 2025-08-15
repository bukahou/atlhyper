package auth

import (
	"NeuroController/db/repository/user"
	"NeuroController/external/uiapi/response"

	"github.com/gin-gonic/gin"
)

// LoginRequest 定义登录请求结构体（接收前端传入的用户名和密码）
type LoginRequest struct {
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码
}

func HandleLogin(c *gin.Context) {
	var req LoginRequest

	// Step 1️⃣: 解析请求体 JSON 数据
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数错误")
		return
	}

	// Step 2️⃣: 查询用户信息
	u, err := user.GetUserByUsername(req.Username)
	if err != nil {
		response.Error(c, "用户不存在")
		return
	}

	// Step 3️⃣: 校验密码
	if !user.CheckPassword(req.Password, u.PasswordHash) {
		response.Error(c, "密码错误")
		return
	}

	// Step 4️⃣: 生成 JWT
	token, err := GenerateToken(u.ID, u.Username, u.Role)
	if err != nil {
		response.ErrorCode(c, 50000, "生成 Token 失败")
		return
	}

	// Step 5️⃣: 登录成功，返回统一结构
	response.Success(c, "登录成功", gin.H{
		"token": token,
		"user": gin.H{
			"id":       u.ID,
			"username": u.Username,
			"displayName": u.DisplayName,
			"role":     u.Role,
		},
	})
}


// =======================================================================
// 📌 GET /auth/user/list
// ✅ 获取所有用户信息（排除密码）
// =======================================================================
func HandleListAllUsers(c *gin.Context) {
	users, err := user.GetAllUsers()
	if err != nil {
		response.Error(c, "获取用户列表失败: "+err.Error())
		return
	}

	response.Success(c, "获取用户列表成功", users)
}


func HandleRegisterUser(c *gin.Context) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Role        int    `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效: "+err.Error())
		return
	}

	userData, err := user.RegisterUser(req.Username, req.Password, req.DisplayName, req.Email, req.Role)
	if err != nil {
		response.ErrorCode(c, 50000, "注册失败: "+err.Error())
		return
	}

	response.Success(c, "✅ 注册成功", userData)
}


func HandleUpdateUserRole(c *gin.Context) {
	var req struct {
		ID   int `json:"id"`
		Role int `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效: "+err.Error())
		return
	}

	if err := user.UpdateUserRole(req.ID, req.Role); err != nil {
		response.ErrorCode(c, 50000, "更新角色失败: "+err.Error())
		return
	}

	response.SuccessMsg(c, "✅ 角色更新成功")
}
