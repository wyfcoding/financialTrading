package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/security"
)

type APIKeyService struct {
	repo domain.APIKeyRepository
}

func NewAPIKeyService(repo domain.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// CreateKey 为用户创建一个新的 API Key。
// 返回 (apiKey, secret, error)。注意：secret 仅在此时返回明文。
func (s *APIKeyService) CreateKey(ctx context.Context, userID, label, scopes string) (*domain.APIKey, string, error) {
	key := fmt.Sprintf("AK%s", idgen.GenShortID(22))
	secret := idgen.GenShortID(32)

	hash, err := security.HashPassword(secret)
	if err != nil {
		return nil, "", err
	}

	ak := &domain.APIKey{
		UserID:     userID,
		Key:        key,
		SecretHash: hash,
		Label:      label,
		Enabled:    true,
		Scopes:     scopes,
	}

	if err := s.repo.Save(ctx, ak); err != nil {
		return nil, "", err
	}

	return ak, secret, nil
}

func (s *APIKeyService) ValidateKey(ctx context.Context, key, secret string) (*domain.APIKey, error) {
	ak, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	if !ak.Enabled {
		return nil, fmt.Errorf("api key disabled")
	}

	if !security.CheckPassword(secret, ak.SecretHash) {
		return nil, fmt.Errorf("invalid api secret")
	}

	return ak, nil
}

func (s *APIKeyService) ListKeys(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	return s.repo.ListByUserID(ctx, userID)
}
