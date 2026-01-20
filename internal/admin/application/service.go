package application

import (
	"context"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/admin/domain"
	// "golang.org/x/crypto/bcrypt" // Mocking for now as we might not have the lib in env
)

type AdminApplicationService struct {
	adminRepo domain.AdminRepository
	roleRepo  domain.RoleRepository
}

func NewAdminApplicationService(adminRepo domain.AdminRepository, roleRepo domain.RoleRepository) *AdminApplicationService {
	return &AdminApplicationService{
		adminRepo: adminRepo,
		roleRepo:  roleRepo,
	}
}

func (s *AdminApplicationService) Login(ctx context.Context, cmd LoginCommand) (*AuthTokenDTO, error) {
	admin, err := s.adminRepo.GetByUsername(ctx, cmd.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Mock Password Check (In reality: bcrypt.CompareHashAndPassword)
	if admin.PasswordHash != cmd.Password { // Naive string compare for mock
		return nil, errors.New("invalid credentials")
	}

	// Mock JWT Generation
	return &AuthTokenDTO{
		Token:     "mock_jwt_token_for_" + admin.Username,
		Type:      "Bearer",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

func (s *AdminApplicationService) CreateAdmin(ctx context.Context, cmd CreateAdminCommand) (uint, error) {
	// Check if role exists
	_, err := s.roleRepo.GetByID(ctx, cmd.RoleID)
	if err != nil {
		return 0, errors.New("role not found")
	}

	// Hash password (mock)
	hashed := cmd.Password // bcrypt.GenerateFromPassword

	admin := domain.NewAdmin(cmd.Username, hashed, cmd.RoleID)
	if err := s.adminRepo.Save(ctx, admin); err != nil {
		return 0, err
	}

	return admin.ID, nil
}

func (s *AdminApplicationService) GetAdmin(ctx context.Context, id uint) (*AdminDTO, error) {
	admin, err := s.adminRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &AdminDTO{
		ID:        admin.ID,
		Username:  admin.Username,
		RoleName:  admin.Role.Name,
		CreatedAt: admin.CreatedAt.Unix(),
	}, nil
}
