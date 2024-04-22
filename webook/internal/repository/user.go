package repository

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/repository/dao"
	"context"
)

var ErrUserDuplicateEmail = dao.ErrUserDuplicateEmail

//var ErrUserDuplicateEmailV1 = fmt.Errorf("%w 邮箱冲突", dao.ErrUserDuplicateEmail)

type UserRepository struct {
	dao *dao.UserDAO
}

func NewUserRepository(dao *dao.UserDAO) *UserRepository {
	return &UserRepository{
		dao: dao,
	}
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) error {
	return r.dao.Insert(ctx, dao.User{
		Email:    u.Email,
		Password: u.Password,
	})
}

func (r *UserRepository) FindBy() {
	// 先从 cache 里面找
	// 再从 dao 里面找
	// 找到了回写 cache

}
