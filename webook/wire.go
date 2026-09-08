//go:build wireinject

package main

import (
	"Clould/webook/internal/repository"
	"Clould/webook/internal/repository/cache"
	"Clould/webook/internal/repository/dao"
	"Clould/webook/internal/service"
	"Clould/webook/internal/web"
	"Clould/webook/ioc"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

func InitWebServer() *gin.Engine {
	// 我只在这里声明我要用的各种东西，但是具体怎么构造，怎么编排顺序
	// 这个方法里面传入各个组件的初始化方法
	wire.Build(
		// 最基础的第三方依赖
		ioc.InitDB, ioc.InitRedis,

		// 初始化 DAO
		dao.NewUserDAO,

		cache.NewUserCache,
		cache.NewCodeCache,

		repository.NewUserRepository,
		repository.NewCodeRepository,

		service.NewUserService,
		service.NewCodeService,
		// 直接基于内存实现

		// 直接基于内存实现
		ioc.InitSMSService,
		web.NewUserHandler,
		// 你中间件呢？
		// 你注册路由呢？
		// 你这个地方没有用到前面的任何东西
		//gin.Default,

		ioc.InitWebServer,
		ioc.InitMiddlewares,
	)
	return new(gin.Engine)
}
