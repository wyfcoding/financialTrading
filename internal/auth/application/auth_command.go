package application

import (
	"context"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
)

// AuthCommandService 认证命令服务
type AuthCommandService struct {
	repo            domain.UserRepository
	apiKeyRepo      domain.APIKeyRepository
	apiKeyRedisRepo domain.APIKeyRedisRepository
	sessionRepo     domain.SessionRepository
	keySvc          *APIKeyService
	publisher       domain.EventPublisher
}

// NewAuthCommandService 创建认证命令服务实例
func NewAuthCommandService(
	repo domain.UserRepository,
	apiKeyRepo domain.APIKeyRepository,
	apiKeyRedisRepo domain.APIKeyRedisRepository,
	sessionRepo domain.SessionRepository,
	keySvc *APIKeyService,
	publisher domain.EventPublisher,
) *AuthCommandService {
	return &AuthCommandService{
		repo:            repo,
		apiKeyRepo:      apiKeyRepo,
		apiKeyRedisRepo: apiKeyRedisRepo,
		sessionRepo:     sessionRepo,
		keySvc:          keySvc,
		publisher:       publisher,
	}
}

// Register 处理用户注册
func (s *AuthCommandService) Register(ctx context.Context, cmd RegisterCommand) (uint, error) {
	// Check if user exists
	_, err := s.repo.GetByEmail(ctx, cmd.Email)
	if err == nil {
		return 0, errors.New("email already registered")
	}

	// Hash password (mock)
	user := domain.NewUser(cmd.Email, cmd.Password)
	if cmd.Role != "" {
		user.Role = cmd.Role
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return 0, err
	}

	// 发布注册事件
	event := domain.UserRegisteredEvent{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "user.registered", cmd.Email, event)

	return user.ID, nil
}

// Login 处理用户登录
func (s *AuthCommandService) Login(ctx context.Context, cmd LoginCommand) (string, int64, error) {
	user, err := s.repo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		return "", 0, errors.New("invalid credentials")
	}

	if user.PasswordHash != cmd.Password { // Mock compare
		return "", 0, errors.New("invalid credentials")
	}

	// 发布登录事件
	event := domain.UserLoggedInEvent{
		UserID:    user.ID,
		Email:     user.Email,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "user.logged_in", cmd.Email, event)

	// 创建会话
	token := "token_" + time.Now().Format("20060102150405") + "_" + cmd.Email
	expTime := time.Now().Add(24 * time.Hour)
	session := &domain.AuthSession{
		Token:     token,
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: time.Now(),
		ExpiresAt: expTime,
	}

	if err := s.sessionRepo.Save(ctx, session); err != nil {
		return "", 0, err
	}

	return token, expTime.Unix(), nil
}

// CreateAPIKey 处理创建API Key
func (s *AuthCommandService) CreateAPIKey(ctx context.Context, cmd CreateAPIKeyCommand) (*domain.APIKey, error) {
	// 使用APIKeyService创建API Key
	key, _, err := s.keySvc.CreateKey(ctx, cmd.UserID, cmd.Label, cmd.Scopes)
	if err != nil {
		return nil, err
	}

	// 获取创建的API Key
	apiKey, err := s.apiKeyRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	// 发布创建事件
	event := domain.APIKeyCreatedEvent{
		APIKeyID:  apiKey.ID,
		UserID:    apiKey.UserID,
		Label:     apiKey.Label,
		Scopes:    apiKey.Scopes,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "api_key.created", apiKey.Key, event)

	// 同步缓存
	_ = s.apiKeyRedisRepo.Save(ctx, apiKey)

	return apiKey, nil
}

// ValidateAPIKey 处理验证API Key
func (s *AuthCommandService) ValidateAPIKey(ctx context.Context, apiKey string) (*domain.APIKey, error) {
	// 尝试从缓存获取
	if cached, err := s.apiKeyRedisRepo.Get(ctx, apiKey); err == nil && cached != nil {
		return cached, nil
	}

	key, err := s.apiKeyRepo.GetByKey(ctx, apiKey)
	if err != nil {
		// 发布验证失败事件
		event := domain.APIKeyValidatedEvent{
			APIKeyID:  0,
			UserID:    "",
			Success:   false,
			Timestamp: time.Now(),
		}
		s.publisher.Publish(ctx, "api_key.validated", apiKey, event)
		return nil, err
	}

	// 发布验证成功事件
	event := domain.APIKeyValidatedEvent{
		APIKeyID:  key.ID,
		UserID:    key.UserID,
		Success:   true,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "api_key.validated", apiKey, event)

	// 同步缓存
	_ = s.apiKeyRedisRepo.Save(ctx, key)

	return key, nil
}

// VerifyAPIKey 处理验证API Key和密钥
func (s *AuthCommandService) VerifyAPIKey(ctx context.Context, key, secret string) (*domain.APIKey, error) {
	apiKey, err := s.keySvc.ValidateKey(ctx, key, secret)
	if err != nil {
		// 发布验证失败事件
		event := domain.APIKeyValidatedEvent{
			APIKeyID:  0,
			UserID:    "",
			Success:   false,
			Timestamp: time.Now(),
		}
		s.publisher.Publish(ctx, "api_key.verified", key, event)
		return nil, err
	}

	// 发布验证成功事件
	event := domain.APIKeyValidatedEvent{
		APIKeyID:  apiKey.ID,
		UserID:    apiKey.UserID,
		Success:   true,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "api_key.verified", key, event)

	return apiKey, nil
}
