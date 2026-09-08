package service

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/repository"
	"context"
	"errors"

	"github.com/redis/go-redis/v9"

	"golang.org/x/crypto/bcrypt"
)

var ErrUserDuplicateEmail = repository.ErrUserDuplicate
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
	return svc.repo.Create(ctx, u)
}

func (svc *UserService) Profile(ctx context.Context,
	id int64) (domain.User, error) {
	u, err := svc.repo.FindById(ctx, id)
	return u, err
}

func (svc *UserService) FindOrCreate(ctx context.Context,
	phone string) (domain.User, error) {
	// 这时候，这个地方要怎么办？
	// 这个叫做快路径
	u, err := svc.repo.FindByPhone(ctx, phone)
	// 要判断，有咩有这个用户
	if err != repository.ErrUserNotFound {
		// 绝大部分请求进来这里
		// nil 会进来这里
		// 不为 ErrUserNotFound 的也会进来这里
		return u, err
	}
	// 在系统资源不足，触发降级之后，不执行慢路径了
	//if ctx.Value("降级") == "true" {
	//	return domain.User{}, errors.New("系统降级了")
	//}
	// 这个叫做慢路径
	// 你明确知道，没有这个用户
	u = domain.User{
		Phone: phone,
	}
	err = svc.repo.Create(ctx, u)
	if err != nil && err != repository.ErrUserDuplicate {
		return u, err
	}
	// 因为这里会遇到主从延迟的问题
	return svc.repo.FindByPhone(ctx, phone)
}
