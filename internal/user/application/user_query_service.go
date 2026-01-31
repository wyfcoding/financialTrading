package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/user/domain"
)

// UserQueryService 用户查询服务
type UserQueryService struct {
	repo domain.UserRepository
}

// NewUserQueryService 创建新的用户查询服务
func NewUserQueryService(repo domain.UserRepository) *UserQueryService {
	return &UserQueryService{
		repo: repo,
	}
}

// GetUserByID 根据ID获取用户
func (s *UserQueryService) GetUserByID(ctx context.Context, id uint) (*domain.UserProfile, error) {
	return s.repo.GetByID(ctx, id)
}

// GetUserByEmail 根据邮箱获取用户
func (s *UserQueryService) GetUserByEmail(ctx context.Context, email string) (*domain.UserProfile, error) {
	// 这里应该实现根据邮箱获取用户的逻辑
	// 假设 repo 中有相应的方法
	// return s.repo.GetByEmail(ctx, email)
	// 暂时返回空实现
	return nil, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserQueryService) GetUserByUsername(ctx context.Context, username string) (*domain.UserProfile, error) {
	// 这里应该实现根据用户名获取用户的逻辑
	// 假设 repo 中有相应的方法
	// return s.repo.GetByUsername(ctx, username)
	// 暂时返回空实现
	return nil, nil
}

// ListUsers 列出所有用户
func (s *UserQueryService) ListUsers(ctx context.Context, page, pageSize int) ([]*domain.UserProfile, int64, error) {
	// 这里应该实现列出所有用户的逻辑，支持分页
	// 假设 repo 中有相应的方法
	// return s.repo.List(ctx, page, pageSize)
	// 暂时返回空实现
	return nil, 0, nil
}
