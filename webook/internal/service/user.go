package service

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/repository"
	"context"

	"golang.org/x/crypto/bcrypt"
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
	// bcrypt md5加密 加盐(随机生成盐值)
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// 存储
	u.Password = string(hash)

	return svc.repo.Create(ctx, u)
}
