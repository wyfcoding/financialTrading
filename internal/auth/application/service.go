package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
)

// AuthApplicationService 认证服务门面，整合命令服务和查询服务
type AuthApplicationService struct {
	commandService *AuthCommandService
	queryService   *AuthQueryService
}

// NewAuthApplicationService 创建认证服务门面实例
func NewAuthApplicationService(
	repo domain.UserRepository,
	apiKeyRepo domain.APIKeyRepository,
	keySvc *APIKeyService,
	publisher domain.EventPublisher,
) *AuthApplicationService {
	return &AuthApplicationService{
		commandService: NewAuthCommandService(repo, apiKeyRepo, keySvc, publisher),
		queryService:   NewAuthQueryService(repo, apiKeyRepo),
	}
}

// Register 处理用户注册
func (s *AuthApplicationService) Register(ctx context.Context, email, password string) (uint, error) {
	cmd := RegisterCommand{
		Email:    email,
		Password: password,
	}
	return s.commandService.Register(ctx, cmd)
}

// Login 处理用户登录
func (s *AuthApplicationService) Login(ctx context.Context, email, password string) (string, int64, error) {
	cmd := LoginCommand{
		Email:    email,
		Password: password,
	}
	return s.commandService.Login(ctx, cmd)
}

// ValidateAPIKey 处理验证API Key
func (s *AuthApplicationService) ValidateAPIKey(ctx context.Context, apiKey string) (*domain.APIKey, error) {
	return s.commandService.ValidateAPIKey(ctx, apiKey)
}

// VerifyAPIKey 处理验证API Key和密钥
func (s *AuthApplicationService) VerifyAPIKey(ctx context.Context, key, secret string) (*domain.APIKey, error) {
	return s.commandService.VerifyAPIKey(ctx, key, secret)
}

// CreateAPIKey 处理创建API Key
func (s *AuthApplicationService) CreateAPIKey(ctx context.Context, userID, label, scopes string, enabled bool) (*domain.APIKey, error) {
	cmd := CreateAPIKeyCommand{
		UserID:  userID,
		Label:   label,
		Scopes:  scopes,
		Enabled: enabled,
	}
	return s.commandService.CreateAPIKey(ctx, cmd)
}

// GetUser 根据ID获取用户信息
func (s *AuthApplicationService) GetUser(ctx context.Context, id uint) (*domain.User, error) {
	return s.queryService.GetUser(ctx, id)
}

// GetUserByEmail 根据邮箱获取用户信息
func (s *AuthApplicationService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.queryService.GetUserByEmail(ctx, email)
}

// ListAPIKeysByUserID 根据用户ID列出API Key
func (s *AuthApplicationService) ListAPIKeysByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	return s.queryService.ListAPIKeysByUserID(ctx, userID)
}
