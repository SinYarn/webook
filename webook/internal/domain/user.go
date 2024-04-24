package domain

import (
	"database/sql"
	"time"
)

// domain 业务概念
// 领域对象 是DDD 中的聚合根中的 entity
// BO(business)
type User struct {
	Id       int64
	Email    string
	Password string
	Nickname string
	// YYYY-MM-DD
	Birthday int64
	AboutMe  string
	// 代表这是一个可以为 NULL 的列
	Phone    sql.NullString
	FileName string
	FileId   int64
	Ctime    time.Time
	Utime    int64
}
type File struct {
	Id         int64
	UserId     int64
	Username   string
	Filename   string
	Filehash   string
	Filesize   int64
	UploadPath string
	Ctime      time.Time
	Utime      int64
}
