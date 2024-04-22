package dao

import "gorm.io/gorm"

/*
对数据库中的表结构进行自动迁移，
如果数据库中不存在名为 users 的表，
则会创建一个名为 users 的表，
并根据 User 结构体的定义自动创建相应的表结构。
*/

// 通过gorm建表
func InitTable(db *gorm.DB) error {
	return db.AutoMigrate(&User{})
}
