package service

import (
	"Clould/internal/domain"
	"Clould/internal/repository"
	"context"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

// Create
func (svc *UserService) SignUp(ctx context.Context, u domain.User) error {
	// 加密放在哪里
	// 存储
	return svc.repo.Create(ctx, u)
}
