package application

import (
	"context"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
)

type AuthApplicationService struct {
	repo       domain.UserRepository
	apiKeyRepo domain.APIKeyRepository
	keySvc     *APIKeyService
}

func NewAuthApplicationService(repo domain.UserRepository, apiKeyRepo domain.APIKeyRepository, keySvc *APIKeyService) *AuthApplicationService {
	return &AuthApplicationService{
		repo:       repo,
		apiKeyRepo: apiKeyRepo,
		keySvc:     keySvc,
	}
}

func (s *AuthApplicationService) Register(ctx context.Context, email, password string) (uint, error) {
	// Check if user exists
	if _, err := s.repo.GetByEmail(ctx, email); err == nil {
		return 0, errors.New("email already registered")
	}
	// Hash password (mock)
	user := domain.NewUser(email, password)
	if err := s.repo.Save(ctx, user); err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (s *AuthApplicationService) Login(ctx context.Context, email, password string) (string, int64, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", 0, errors.New("invalid credentials")
	}
	if user.PasswordHash != password { // Mock compare
		return "", 0, errors.New("invalid credentials")
	}
	exp := time.Now().Add(24 * time.Hour).Unix()
	return "mock_jwt_" + email, exp, nil
}

func (s *AuthApplicationService) ValidateAPIKey(ctx context.Context, apiKey string) (*domain.APIKey, error) {
	return s.apiKeyRepo.GetByKey(ctx, apiKey)
}

func (s *AuthApplicationService) VerifyAPIKey(ctx context.Context, key, secret string) (*domain.APIKey, error) {
	return s.keySvc.ValidateKey(ctx, key, secret)
}
