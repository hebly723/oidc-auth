package handler

import (
	"github.com/gin-gonic/gin"
)

// generateInviteCodeHandler 生成邀请码处理器方法
func (s *Server) generateInviteCodeHandler(c *gin.Context) {
	if s.InviteCodeHandler == nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "Invite code service not initialized",
		})
		return
	}
	s.InviteCodeHandler.GenerateInviteCode(c)
}

// getInviteCodesHandler 获取邀请码列表处理器方法
func (s *Server) getInviteCodesHandler(c *gin.Context) {
	if s.InviteCodeHandler == nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "Invite code service not initialized",
		})
		return
	}
	s.InviteCodeHandler.GetInviteCodes(c)
}

// processInviteCodeInLogin 在登录过程中处理邀请码
func (s *Server) processInviteCodeInLogin(c *gin.Context, inviteCode string, userID string, user interface{}) error {
	if s.InviteCodeHandler == nil || inviteCode == "" {
		return nil // 如果没有邀请码处理器或邀请码为空，跳过处理
	}
	
	// 这里需要类型断言，但为了避免循环依赖，我们在login_handler中直接调用
	// 实际的邀请码验证逻辑在InviteCodeHandler.ValidateInviteCode中实现
	return nil
}