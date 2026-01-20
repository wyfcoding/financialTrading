package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/user/domain"
)

type UserApplicationService struct{ repo domain.UserRepository }

func NewUserApplicationService(repo domain.UserRepository) *UserApplicationService {
	return &UserApplicationService{repo: repo}
}

func (s *UserApplicationService) GetUser(ctx context.Context, id uint) (*domain.UserProfile, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserApplicationService) UpdateUser(ctx context.Context, id uint, name, phone string) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	u.Name = name
	u.Phone = phone
	return s.repo.Save(ctx, u)
}
