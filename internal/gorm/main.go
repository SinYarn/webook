package main

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Product struct {
	// 软删除方式
	gorm.Model
	Code  string
	Price uint
}

func main() {
	// 创建一个db，数据库初始化
	// sqlite基于内存
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		// 只输出语句，不执行
		DryRun: true,
	})

	// 连接mysql数据库， dsn：用户：密码@tcp(数据库主机ip：3306端口)
	//db, err = gorm.Open(mysql.Open("root:root@tcp(localhost:3306)/your_db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db = db.Debug()
	// 迁移 schema
	// 建表
	db.AutoMigrate(&Product{})

	// Create
	db.Create(&Product{Code: "D42", Price: 100})

	// Read
	var product Product
	db.First(&product, 1)                 // 根据整型主键查找
	db.First(&product, "code = ?", "D42") // 查找 code 字段值为 D42 的记录

	// Update - 将 product 的 price 更新为 200
	db.Model(&product).Update("Price", 200)
	// Update - 更新多个字段
	db.Model(&product).Updates(Product{Price: 200, Code: "F42"}) // 仅更新非零值字段
	db.Model(&product).Updates(map[string]interface{}{"Price": 200, "Code": "F42"})

	// Delete - 删除 product
	db.Delete(&product, 1)
}
