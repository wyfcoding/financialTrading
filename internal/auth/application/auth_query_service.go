package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
)

// AuthQueryService 认证查询服务
type AuthQueryService struct {
	repo       domain.UserRepository
	apiKeyRepo domain.APIKeyRepository
}

// NewAuthQueryService 创建认证查询服务实例
func NewAuthQueryService(
	repo domain.UserRepository,
	apiKeyRepo domain.APIKeyRepository,
) *AuthQueryService {
	return &AuthQueryService{
		repo:       repo,
		apiKeyRepo: apiKeyRepo,
	}
}

// GetUser 根据ID获取用户信息
func (s *AuthQueryService) GetUser(ctx context.Context, id uint) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

// GetUserByEmail 根据邮箱获取用户信息
func (s *AuthQueryService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.repo.GetByEmail(ctx, email)
}

// GetAPIKey 根据Key获取API Key信息
func (s *AuthQueryService) GetAPIKey(ctx context.Context, key string) (*domain.APIKey, error) {
	return s.apiKeyRepo.GetByKey(ctx, key)
}

// ListAPIKeysByUserID 根据用户ID列出API Key
func (s *AuthQueryService) ListAPIKeysByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	return s.apiKeyRepo.ListByUserID(ctx, userID)
}
