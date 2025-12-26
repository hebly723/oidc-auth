package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/zgsm-ai/oidc-auth/internal/repository"
	"github.com/zgsm-ai/oidc-auth/pkg/errs"
	"github.com/zgsm-ai/oidc-auth/pkg/response"
)

type UpdateVipRequest struct {
	UserID    string     `json:"user_id" binding:"required"`
	Vip       int        `json:"vip"`
	VipExpire *time.Time `json:"vip_expire"`
}

// updateVipHandler handles the function to update user VIP information
func (s *Server) updateVipHandler(c *gin.Context) {
	var req UpdateVipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, errs.ErrBadRequestParam, err.Error())
		return
	}

	// Validate user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, errs.ErrBadRequestParam, "Invalid user ID")
		return
	}

	ctx, cancel := getContextWithTimeout(defaultTimeout)
	defer cancel()

	// Get user information
	user, err := repository.GetDB().GetUserByField(ctx, "id", userID)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, errs.ErrUserNotFound, "Failed to query user")
		return
	}

	if user == nil {
		response.JSONError(c, http.StatusNotFound, errs.ErrUserNotFound, "User does not exist")
		return
	}

	// Update VIP information
	user.Vip = req.Vip
	user.VipExpire = req.VipExpire
	user.UpdatedAt = time.Now()

	// Save the update
	err = repository.GetDB().Upsert(ctx, user, "id", user.ID)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, errs.ErrUpdateInfo, "Failed to update user VIP information")
		return
	}

	response.JSONSuccess(c, "VIP information updated successfully", gin.H{
		"user_id":    user.ID.String(),
		"vip":        user.Vip,
		"vip_expire": user.VipExpire,
	})
}
