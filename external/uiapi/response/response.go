package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 通用响应结构体
type Response struct {
	Code    int         `json:"code"`             // 状态码（前端识别）
	Message string      `json:"message,omitempty"`// 提示信息
	Data    interface{} `json:"data,omitempty"`   // 返回数据
}

// 成功响应（默认 code=20000）
func Success(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    20000,
		Message: message,
		Data:    data,
	})
}

// 成功但无需数据
func SuccessMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{
		Code:    20000,
		Message: msg,
	})
}

// 错误响应（如登录失败、无权限等，默认 code=40000）
func Error(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{
		Code:    40000,
		Message: msg,
	})
}

// 自定义错误码（如 code=50000 内部错误）
func ErrorCode(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: msg,
	})
}
