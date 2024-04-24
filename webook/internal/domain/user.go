package domain

import "time"

// domain 业务概念
// 领域对象 是DDD 中的聚合根中的 entity
// BO(business)
type User struct {
	Id       int64
	Email    string
	Password string
	Ctime    time.Time
}
