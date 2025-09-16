package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zgsm-ai/oidc-auth/internal/repository"
	"github.com/zgsm-ai/oidc-auth/internal/service"
	"github.com/zgsm-ai/oidc-auth/pkg/log"
)

// InviteCodeHandler 邀请码处理器
type InviteCodeHandler struct {
	inviteCodeSvc *service.InviteCodeService
	db            *repository.Database
}

// NewInviteCodeHandler 创建邀请码处理器
func NewInviteCodeHandler(inviteCodeSvc *service.InviteCodeService, db *repository.Database) *InviteCodeHandler {
	return &InviteCodeHandler{
		inviteCodeSvc: inviteCodeSvc,
		db:            db,
	}
}

// GenerateInviteCodeRequest 生成邀请码请求
type GenerateInviteCodeRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// GenerateInviteCodeResponse 生成邀请码响应
type GenerateInviteCodeResponse struct {
	Code      string `json:"code"`
	CreatedAt string `json:"created_at"`
}

// GetInviteCodesResponse 获取邀请码列表响应
type GetInviteCodesResponse struct {
	Codes []InviteCodeInfo `json:"codes"`
}

// InviteCodeInfo 邀请码信息
type InviteCodeInfo struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	CreatedAt string `json:"created_at"`
}

// GenerateInviteCode 生成邀请码
func (h *InviteCodeHandler) GenerateInviteCode(c *gin.Context) {
	var req GenerateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error(c.Request.Context(), "Invalid request parameters: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request parameters",
			"error":   err.Error(),
		})
		return
	}

	// 验证用户ID格式
	if strings.TrimSpace(req.UserID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "User ID cannot be empty",
		})
		return
	}

	// 验证用户是否存在
	user, err := h.db.GetUserByField(c.Request.Context(), "id", req.UserID)
	if err != nil {
		log.Error(c.Request.Context(), "Failed to get user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal server error",
		})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "User not found",
		})
		return
	}

	// 生成邀请码
	inviteCode, err := h.inviteCodeSvc.GenerateInviteCode(c.Request.Context(), req.UserID)
	if err != nil {
		log.Error(c.Request.Context(), "Failed to generate invite code: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to generate invite code",
		})
		return
	}

	// 返回邀请码
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Invite code generated successfully",
		"data": GenerateInviteCodeResponse{
			Code:      inviteCode.Code,
			CreatedAt: inviteCode.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// GetInviteCodes 获取用户的邀请码列表
func (h *InviteCodeHandler) GetInviteCodes(c *gin.Context) {
	userID := c.Query("user_id")
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "User ID is required",
		})
		return
	}

	// 验证用户是否存在
	user, err := h.db.GetUserByField(c.Request.Context(), "id", userID)
	if err != nil {
		log.Error(c.Request.Context(), "Failed to get user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal server error",
		})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "User not found",
		})
		return
	}

	// 获取邀请码列表
	codes, err := h.inviteCodeSvc.GetUserInviteCodes(c.Request.Context(), userID)
	if err != nil {
		log.Error(c.Request.Context(), "Failed to get invite codes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get invite codes",
		})
		return
	}

	// 转换为响应格式
	var codeInfos []InviteCodeInfo
	for _, code := range codes {
		codeInfos = append(codeInfos, InviteCodeInfo{
			ID:        code.ID.String(),
			Code:      code.Code,
			CreatedAt: code.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data": GetInviteCodesResponse{
			Codes: codeInfos,
		},
	})
}

// ValidateInviteCode 验证邀请码（内部使用）
func (h *InviteCodeHandler) ValidateInviteCode(c *gin.Context, inviteCode string, userID string, user interface{}) error {
	if strings.TrimSpace(inviteCode) == "" {
		return nil // 邀请码为空，跳过验证
	}

	// 类型断言，将interface{}转换为*repository.AuthUser
	authUser, ok := user.(*repository.AuthUser)
	if !ok && user != nil {
		log.Error(c.Request.Context(), "Invalid user type for invite code validation")
		return fmt.Errorf("invalid user type")
	}

	// 验证并使用邀请码
	_, err := h.inviteCodeSvc.ValidateAndUseInviteCode(c.Request.Context(), inviteCode, userID, authUser)
	if err != nil {
		log.Error(c.Request.Context(), "Failed to validate invite code: %v", err)
		return err
	}

	log.Info(c.Request.Context(), "Successfully processed invite code %s for user %s", inviteCode, userID)
	return nil
}
