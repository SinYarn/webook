package dao

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"gorm.io/gorm"
)

var (
	ErrUserDuplicateEmail = errors.New("用户邮箱冲突")
	ErrUserNotFound       = gorm.ErrRecordNotFound
)

type UserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		db: db,
	}
}

type FileDAO struct {
	db *gorm.DB
}

func NewFileDAO(db *gorm.DB) *FileDAO {
	return &FileDAO{
		db: db,
	}
}

func (dao *UserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	return u, err
}

func (dao *UserDAO) Insert(ctx context.Context, u User) error {
	// 存毫秒数
	now := time.Now().UnixMilli()
	u.Utime = now
	u.Ctime = now

	// 错误码
	err := dao.db.WithContext(ctx).Create(&u).Error
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		// mysql 唯一索引错误码 ok: 确实是mysql的错误
		const uniqueConflictsErrNo uint16 = 1062
		if mysqlErr.Number == uniqueConflictsErrNo {
			// 邮箱冲突
			return ErrUserDuplicateEmail
		}
	}
	return err
}

func (dao *FileDAO) Upload(ctx context.Context, f File) error {
	// 存毫秒数
	now := time.Now().UnixMilli()
	f.Utime = now
	f.Ctime = now

	// 错误码
	err := dao.db.WithContext(ctx).Create(&f).Error
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		// mysql 唯一索引错误码 ok: 确实是mysql的错误
		const uniqueConflictsErrNo uint16 = 1062
		if mysqlErr.Number == uniqueConflictsErrNo {
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
	Nickname string `gorm:"type=varchar(128)"`
	// YYYY-MM-DD
	Birthday int64
	AboutMe  string `gorm:"type=varchar(4096)"`

	// 代表这是一个可以为 NULL 的列
	Phone    sql.NullString `gorm:"unique"`
	FileName string
	FileId   int64
	// 创建时间, 毫秒数
	Ctime int64
	// 更新时间, 毫秒数
	Utime int64
}

type File struct {
	Id         int64 `gorm:"primarykey, autoIncrement"`
	UserId     int64
	Username   string
	Filename   string
	Filehash   string
	Filesize   int64
	UploadPath string
	// 创建时间, 毫秒数
	Ctime int64
	// 更新时间, 毫秒数
	Utime int64
}
