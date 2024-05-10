package main

import (
	"Clould/webook/config"
	"Clould/webook/internal/repository"
	dao2 "Clould/webook/internal/repository/dao"
	"Clould/webook/internal/service"
	"Clould/webook/internal/web"
	"Clould/webook/internal/web/middleware"
	"Clould/webook/pkg/ginx/middlewares/ratelimit"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化数据库
	db := initDB()
	// 2. 初始化web服务
	server := initWebServer()
	// 3. 初始化 DDD分层结构
	u := initUser(db)
	// 4. 注册分组路由
	u.RegisterRoutes(server)

	/*	server.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello start Kubernetes!")
	})*/

	// 监听并在 0.0.0.0:8080 上启动服务
	server.Run(":8080")
}

// 1. 初始化数据库
func initDB() *gorm.DB {
	// 链接mysql数据库(docker中)
	db, err := gorm.Open(mysql.Open(config.Config.DB.DSN))
	if err != nil {
		// panic: go goroutine 结束
		// 初始化错误, 应用不启动
		panic(err)
	}

	// 数据库建表
	err = dao2.InitTable(db)
	if err != nil {
		// 建表不成功, 终止应用
		panic(err)
	}

	return db
}

// 2. 初始化web服务
func initWebServer() *gin.Engine {
	server := gin.Default()

	// middleware中间件, 在request之前执行
	server.Use(func(ctx *gin.Context) {
		println("这是第一个 middleware")
	})

	server.Use(func(ctx *gin.Context) {
		println("这是第二个 middleware")
	})

	// 初始化redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Config.Redis.Addr,
		Password: "",
		DB:       1,
	})

	// 使用限流插件 1秒100个请求
	server.Use(ratelimit.NewBuilder(redisClient, time.Second, 100).Build())

	// middleware中间件: 解决跨域请求的问题
	server.Use(cors.New(cors.Config{
		// AllowOrigins: []string{"http://localhost:3000"},
		// AllowMethods: []string{"POST", "GET"},
		AllowHeaders: []string{"Content-Type", "Authorization"},

		// 允许正式请求，响应带的header, 加这个前端才能拿得到
		ExposeHeaders: []string{"x-jwt-token"},

		// 是否允许带 cookie 之类的
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				// 你的开发环境
				return true
			}
			// 其他开发环境：你的域名
			return strings.Contains(origin, "live.webook.com")
		},
		MaxAge: 12 * time.Hour,
	}))

	// 方式1: 使用cookie
	// 登录校验模块: 初始化
	// 登录校验模块步骤1: 生成session, 放在cookie里
	// store := cookie.NewStore([]byte("secret"))

	// 方式2: 基于内存的
	// store := memstore.NewStore([]byte("k6CswdUm75WKcbM68UQUuxVsHSpTCwgK"), []byte("eF1`yQ9>yT1`tH1,sJ0.zD8;mZ9~nC6("))

	// 方式3: 基于redis 主流
	// docker redis 端口:6379 密码为空

	//store, err := redis.NewStore(16, "tcp", "localhost:6379", "",
	//	[]byte("k6CswdUm75WKcbM68UQUuxVsHSpTCwgK"), []byte("eF1`yQ9>yT1`tH1,sJ0.zD8;mZ9~nC6("))
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	// session存在context中, context中
	// server.Use(sessions.Sessions("mysession", store)) // 设置 我的session的名字是 mysession 到cookie中,
	// session存在context中

	// 校验模块步骤3: 使用 验证session
	// v1
	// session 校验
	/*
		server.Use(middleware.NewLoginMiddlewareBuilder().
		IgnorePaths("/users/signup").
		IgnorePaths("/users/login").Build()) //  链式调用
	*/

	// jwt 校验
	server.Use(middleware.NewLoginJWTMiddlewareBuilder().
		IgnorePaths("/users/signup").
		IgnorePaths("/users/login").Build())

	return server
}

// 3. 初始化 DDD分层结构
func initUser(db *gorm.DB) *web.UserHandler {
	// DDD架构
	ud := dao2.NewUserDAO(db)
	repo := repository.NewUserRepository(ud)
	svc := service.NewUserService(repo)
	// 预编译 正则表达式（邮箱、 密码匹配） -- 优化项目性能, 提高校验速度
	u := web.NewUserHandler(svc)
	return u
}
