package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
)

// AuthQueryService 认证查询服务
type AuthQueryService struct {
	repo            domain.UserRepository
	apiKeyRepo      domain.APIKeyRepository
	apiKeyRedisRepo domain.APIKeyRedisRepository
	sessionRepo     domain.SessionRepository
}

// NewAuthQueryService 创建认证查询服务实例
func NewAuthQueryService(
	repo domain.UserRepository,
	apiKeyRepo domain.APIKeyRepository,
	apiKeyRedisRepo domain.APIKeyRedisRepository,
	sessionRepo domain.SessionRepository,
) *AuthQueryService {
	return &AuthQueryService{
		repo:            repo,
		apiKeyRepo:      apiKeyRepo,
		apiKeyRedisRepo: apiKeyRedisRepo,
		sessionRepo:     sessionRepo,
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

// GetAPIKey 根据Key获取API Key信息（优先缓存）
func (s *AuthQueryService) GetAPIKey(ctx context.Context, key string) (*domain.APIKey, error) {
	if cached, err := s.apiKeyRedisRepo.Get(ctx, key); err == nil && cached != nil {
		return cached, nil
	}

	ak, err := s.apiKeyRepo.GetByKey(ctx, key)
	if err == nil && ak != nil {
		_ = s.apiKeyRedisRepo.Save(ctx, ak)
	}
	return ak, err
}

// ListAPIKeysByUserID 根据用户ID列出API Key
func (s *AuthQueryService) ListAPIKeysByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	return s.apiKeyRepo.ListByUserID(ctx, userID)
}

// GetSession 根据 Token 获取会话
func (s *AuthQueryService) GetSession(ctx context.Context, token string) (*domain.AuthSession, error) {
	return s.sessionRepo.Get(ctx, token)
}
