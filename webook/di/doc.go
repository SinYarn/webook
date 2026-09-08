// Copyright 2020 Baidu Inc. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

/*
modification history
--------------------
2026/9/8, by v_fangxiang01@baidu.com, create
*/

package di

import (
	"Clould/webook/config"
	"Clould/webook/internal/repository/dao"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 1. 初始化数据库
func InitDB() *gorm.DB {
	// 链接mysql数据库(docker中)
	db, err := gorm.Open(mysql.Open(config.Config.DB.DSN))
	if err != nil {
		// panic: go goroutine 结束
		// 初始化错误, 应用不启动
		panic(err)
	}

	// 数据库建表
	err = dao.InitTable(db)
	if err != nil {
		// 建表不成功, 终止应用
		panic(err)
	}

	return db
}
