package middleware

import (
	"github.com/gin-gonic/gin"
)

// InviteCodeHandler 邀请码处理器接口
// 这个接口定义了邀请码处理器需要实现的方法
type InviteCodeHandler interface {
	GenerateInviteCode(c *gin.Context)
	GetInviteCodes(c *gin.Context)
	ValidateInviteCode(c *gin.Context, inviteCode string, userID string, user interface{}) error
}

// InjectInviteCodeHandler 注入邀请码处理器到context中
// 使用interface{}来避免循环依赖，在运行时进行类型检查
func InjectInviteCodeHandler(inviteCodeHandler interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if inviteCodeHandler != nil {
			c.Set("inviteCodeHandler", inviteCodeHandler)
		}
		c.Next()
	}
}

// GetInviteCodeHandler 从context中获取邀请码处理器
func GetInviteCodeHandler(c *gin.Context) InviteCodeHandler {
	if handler, exists := c.Get("inviteCodeHandler"); exists {
		if inviteHandler, ok := handler.(InviteCodeHandler); ok {
			return inviteHandler
		}
	}
	return nil
}
