package repository

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/repository/cache"
	"Clould/webook/internal/repository/dao"
	"context"
)

var (
	ErrUserDuplicateEmail = dao.ErrUserDuplicateEmail
	ErrUserNotFound       = dao.ErrUserNotFound
)

//var ErrUserDuplicateEmailV1 = fmt.Errorf("%w 邮箱冲突", dao.ErrUserDuplicateEmail)

type UserRepository struct {
	dao   *dao.UserDAO
	cache *cache.UserCache
}

func NewUserRepository(dao *dao.UserDAO, c *cache.UserCache) *UserRepository {
	return &UserRepository{
		dao:   dao,
		cache: c,
	}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := r.dao.FindByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{
		Id:       u.Id,
		Email:    u.Email,
		Password: u.Password,
	}, nil
}

func (ur *UserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
	// 先从 cache 里面找
	u, err := ur.cache.Get(ctx, id)
	// 缓存有数据
	if err == nil {
		return u, nil
	}
	// 缓存没有数据 err == redis.nil
	/*	if err == cache.ErrKeyNotExist {
		// 去数据库中加载
		// 再从 dao 里面找
		//找到了回写 cache
	}*/
	ue, err := ur.dao.FindById(ctx, id)
	if err != nil {
		return domain.User{}, err
	}

	u = domain.User{
		Id:       ue.Id,
		Email:    ue.Email,
		Password: ue.Password,
	}
	// 设置 go runtime
	go func() {
		err = ur.cache.Set(ctx, u)
		if err != nil {
			// 缓存设置失败
			// 我这里怎么办
			// 打日志做监控
		}
	}()
	return u, err
	// redis 数据库限流 每秒200qps 布隆过滤器
	// 缓存出错了
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) error {
	return r.dao.Insert(ctx, dao.User{
		Email:    u.Email,
		Password: u.Password,
	})
}
