package dao

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"gorm.io/gorm"
)

var (
	ErrUserDuplicateEmail = errors.New("用户邮箱冲突")
)

type UserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		db: db,
	}
}

func (dao *UserDAO) Insert(ctx context.Context, u User) error {
	// 存毫秒数
	now := time.Now().UnixMilli()
	u.Utime = now
	u.Ctime = now

	// 错误码
	err := dao.db.WithContext(ctx).Create(&u).Error
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		// mysql 唯一索引错误码
		const uniqueConflictsErrNo uint16 = 1062
		if mysqlErr.Number == 1062 {
			// 邮箱冲突
			return ErrUserDuplicateEmail
		}
	}
	return err
}

// User 直接对应数据库结构
// 数据库层面的	entity model PO (persistent object)
type User struct {
	Id       int64  `gorm:"primarykey, autoIncrement"`
	Email    string `gorm:"unique"`
	Password string

	// 创建时间, 毫秒数
	Ctime int64
	// 更新时间, 毫秒数
	Utime int64
}

type UserDetail struct {
}
