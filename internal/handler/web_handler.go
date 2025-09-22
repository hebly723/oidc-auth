package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgsm-ai/oidc-auth/internal/constants"
	"github.com/zgsm-ai/oidc-auth/internal/providers"
	"github.com/zgsm-ai/oidc-auth/internal/repository"
	"github.com/zgsm-ai/oidc-auth/pkg/errs"
	"github.com/zgsm-ai/oidc-auth/pkg/response"
	"github.com/zgsm-ai/oidc-auth/pkg/utils"
)

// WebParameterCarrier carries web login parameters through the OAuth flow
type WebParameterCarrier struct {
	Provider string `json:"provider"`
}

// webLoginHandler handles web login requests
func (s *Server) webLoginHandler(c *gin.Context) {
	provider := c.DefaultQuery("provider", "casdoor")

	oauthManager := providers.GetManager()
	providerInstance, err := oauthManager.GetProvider(provider)
	if err != nil {
		response.HandleError(c, http.StatusInternalServerError, errs.ErrBadRequestParam, err)
		return
	}

	// Use empty state parameter, invite code will be returned through state after user input on Casdoor page
	state := ""

	authURL := providerInstance.GetAuthURL(state, s.BaseURL+constants.WebLoginCallbackURI)

	response.JSONSuccess(c, "", map[string]interface{}{
		"state": state,
		"url":   authURL,
	})
}

// webLoginCallbackHandler handles web login callback with invite code processing
func (s *Server) webLoginCallbackHandler(c *gin.Context) {
	code := c.DefaultQuery("code", "")
	inviterCode := c.DefaultQuery("inviter_code", "")
	if inviterCode == "" {
		inviterCode = c.DefaultQuery("state", "")
	}

	if code == "" {
		response.JSONError(c, http.StatusBadRequest, errs.ErrBadRequestParam,
			errs.ParamNeedErr("code").Error())
		return
	}

	provider := "casdoor" // Fixed to use casdoor

	oauthManager := providers.GetManager()
	providerInstance, err := oauthManager.GetProvider(provider)
	if err != nil {
		response.HandleError(c, http.StatusBadRequest, errs.ErrBadRequestParam, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Get user info from OAuth provider and validate inviter code
	user, err := GetWebUserByOauth(ctx, code, provider, inviterCode)
	if err != nil {
		response.HandleError(c, http.StatusInternalServerError, errs.ErrUserNotFound,
			fmt.Errorf("%s: %v", errs.ErrInfoQueryUserInfo, err))
		return
	}

	if user == nil {
		response.HandleError(c, http.StatusUnauthorized, errs.ErrTokenInvalid, errs.ErrInfoInvalidToken)
		return
	}

	// Update or create user with invite code processing
	err = providerInstance.Update(ctx, user)
	if err != nil {
		response.HandleError(c, http.StatusInternalServerError, errs.ErrUpdateInfo,
			fmt.Errorf("%s: %v", errs.ErrInfoUpdateUserInfo, err))
		return
	}

	// Redirect to bind account page
	c.Redirect(http.StatusFound, providerInstance.GetEndpoint(false)+constants.BindAccountBindURI)
}

// GetWebUserByOauth gets user info from OAuth provider and processes inviter code for web login
func GetWebUserByOauth(ctx context.Context, code, provider, inviterCode string) (*repository.AuthUser, error) {
	oauthManager := providers.GetManager()
	providerInstance, err := oauthManager.GetProvider(provider)
	if err != nil {
		return nil, err
	}

	// Exchange code for token
	token, err := providerInstance.ExchangeToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %v", err)
	}

	// Get user info from provider
	user, err := providerInstance.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", errs.ErrInfoQueryUserInfo, err)
	}

	// Check if this is a new user (first time login)
	var existingUser *repository.AuthUser
	if user.GithubID != "" {
		existingUser, err = repository.GetDB().GetUserByField(ctx, "github_id", user.GithubID)
	} else if user.Phone != "" {
		existingUser, err = repository.GetDB().GetUserByField(ctx, "phone", user.Phone)
	} else if user.Email != "" {
		existingUser, err = repository.GetDB().GetUserByField(ctx, "email", user.Email)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Handle inviter code validation based on user status
	if inviterCode != "" {
		if existingUser != nil {
			// Existing user cannot use inviter code
			return nil, fmt.Errorf("you have registered")
		}

		// New user with inviter code - validate and set inviter ID
		inviter, err := utils.ValidateInviteCode(ctx, inviterCode)
		if err != nil {
			return nil, fmt.Errorf("invalid inviter code: %w", err)
		}
		user.InviterID = inviter.ID
	}

	// Note: User's own invite code will be generated when they first access the invite-code endpoint
	return user, nil
}

// getUserInviteCodeHandler gets current user's invite code
func (s *Server) getUserInviteCodeHandler(c *gin.Context) {
	// Get token from request header
	token, err := getTokenFromHeader(c)
	if err != nil {
		response.JSONError(c, http.StatusUnauthorized, errs.ErrAuthentication, "authentication failed: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user info by access token
	user, _, err := utils.GetUserByTokenHash(ctx, token, "access_token_hash")
	if err != nil {
		response.JSONError(c, http.StatusUnauthorized, errs.ErrUserNotFound, "user not found: "+err.Error())
		return
	}

	if user == nil {
		response.JSONError(c, http.StatusUnauthorized, errs.ErrUserNotFound, "user not found")
		return
	}

	// Generate invite code if user doesn't have one
	if user.InviteCode == "" {
		inviteCode, err := utils.GenerateUniqueInviteCode(ctx)
		if err != nil {
			response.JSONError(c, http.StatusInternalServerError, errs.ErrUpdateInfo, "failed to get invite code: "+err.Error())
			return
		}

		// Update user's invite code
		user.InviteCode = inviteCode
		user.UpdatedAt = time.Now()

		// Update user record with new invite code
		err = repository.GetDB().Upsert(ctx, user, "id", user.ID)
		if err != nil {
			response.JSONError(c, http.StatusInternalServerError, errs.ErrUpdateInfo, "failed to update invite code: "+err.Error())
			return
		}
	}

	// Return user invite code information
	response.JSONSuccess(c, "", gin.H{
		"invite_code": user.InviteCode,
	})
}
