package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/zgsm-ai/oidc-auth/internal/constants"
	"github.com/zgsm-ai/oidc-auth/internal/repository"
	"github.com/zgsm-ai/oidc-auth/pkg/log"
)

// InviteCodeService 邀请码服务
type InviteCodeService struct {
	db *repository.Database
}

// NewInviteCodeService 创建邀请码服务实例
func NewInviteCodeService(db *repository.Database) *InviteCodeService {
	return &InviteCodeService{
		db: db,
	}
}

// GenerateInviteCode 生成邀请码
func (s *InviteCodeService) GenerateInviteCode(ctx context.Context, userID string) (*repository.InviteCode, error) {
	// 生成随机邀请码
	code, err := s.generateRandomCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random code: %w", err)
	}

	// 检查邀请码是否已存在
	existingCode, err := s.db.GetInviteCodeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invite code: %w", err)
	}

	// 如果邀请码已存在，重新生成
	if existingCode != nil {
		return s.GenerateInviteCode(ctx, userID)
	}

	// 创建邀请码记录
	inviteCode := &repository.InviteCode{
		ID:        uuid.New(),
		Code:      code,
		UserID:    uuid.MustParse(userID),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.CreateInviteCode(ctx, inviteCode); err != nil {
		return nil, fmt.Errorf("failed to create invite code: %w", err)
	}

	log.Info(ctx, "Generated invite code %s for user %s", code, userID)
	return inviteCode, nil
}

// ValidateAndUseInviteCode 验证并使用邀请码
func (s *InviteCodeService) ValidateAndUseInviteCode(ctx context.Context, code string, userID string, user *repository.AuthUser) (*repository.InviteCodeUsage, error) {
	// 1. 检查邀请码是否存在
	inviteCode, err := s.db.GetInviteCodeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite code: %w", err)
	}
	if inviteCode == nil {
		return nil, fmt.Errorf("invite code not found")
	}

	// 2. 检查邀请码是否在有效期内
	if time.Since(inviteCode.CreatedAt) > time.Duration(constants.InviteCodeValidDays)*24*time.Hour {
		return nil, fmt.Errorf("invite code expired")
	}

	// 3. 检查用户是否在注册后的有效期内
	if time.Since(user.CreatedAt) > time.Duration(constants.UserRegisterValidDays)*24*time.Hour {
		return nil, fmt.Errorf("user registration period expired for using invite code")
	}

	// 4. 检查用户是否已使用过其他邀请码
	existingUsage, err := s.db.GetInviteCodeUsageByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invite code usage: %w", err)
	}
	if existingUsage != nil {
		return nil, fmt.Errorf("user has already used an invite code")
	}

	// 5. 创建邀请码使用记录
	usage := &repository.InviteCodeUsage{
		ID:           uuid.New(),
		InviteCodeID: inviteCode.ID,
		UserID:       uuid.MustParse(userID),
		UsedAt:       time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.db.CreateInviteCodeUsage(ctx, usage); err != nil {
		return nil, fmt.Errorf("failed to create invite code usage: %w", err)
	}

	log.Info(ctx, "User %s used invite code %s", userID, code)
	return usage, nil
}

// GetUserInviteCodes 获取用户的邀请码列表
func (s *InviteCodeService) GetUserInviteCodes(ctx context.Context, userID string) ([]repository.InviteCode, error) {
	codes, err := s.db.GetInviteCodesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user invite codes: %w", err)
	}
	return codes, nil
}

// generateRandomCode 生成随机邀请码
func (s *InviteCodeService) generateRandomCode() (string, error) {
	const charset = constants.InviteCodeChars
	const length = constants.InviteCodeLength

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}
