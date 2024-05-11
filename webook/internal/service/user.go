package service

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"golang.org/x/crypto/bcrypt"
)

var ErrUserDuplicateEmail = repository.ErrUserDuplicateEmail
var ErrInvalidUserOrPassword = errors.New("账号/邮箱或者密码不正确")

type UserService struct {
	repo  *repository.UserRepository
	redis *redis.Client
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (svc *UserService) Login(ctx context.Context, email string, password string) (domain.User, error) {
	// 通过邮箱查mysql的密码
	u, err := svc.repo.FindByEmail(ctx, email)
	if err == repository.ErrUserNotFound {
		// 邮箱不存在
		return domain.User{}, ErrInvalidUserOrPassword
	}
	if err != nil {
		// 数据库超时
		return domain.User{}, err
	}
	// 比较密码    加密后 : 前端传过来
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		// DEBUG 账号密码不匹配
		return domain.User{}, ErrInvalidUserOrPassword
	}
	return u, nil
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

	// 然后就是存起来
	err := svc.repo.Create(ctx, u)
	if err != nil {
		return err
	}

	return svc.repo.Create(ctx, u)
}

func (svc *UserService) Profile(ctx context.Context,
	id int64) (domain.User, error) {
	// 第一个念头
	val, err := svc.redis.Get(ctx, fmt.Sprintf("user:%d", id)).Result()
	if err != nil {
		return domain.User{}, err
	}

}
