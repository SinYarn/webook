package main

import (
	"Clould/webook/internal/repository"
	dao2 "Clould/webook/internal/repository/dao"
	"Clould/webook/internal/service"
	"Clould/webook/internal/web"
	"Clould/webook/internal/web/middleware"
	"strings"
	"time"

	"github.com/gin-contrib/sessions/redis"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
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

	// 监听并在 0.0.0.0:8080 上启动服务
	server.Run()
}

// 1. 初始化数据库
func initDB() *gorm.DB {
	// 链接mysql数据库(docker中)
	db, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13316)/webook"))
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
		println("这是第一个middleware")
	})

	// middleware中间件: 解决跨域请求的问题
	server.Use(cors.New(cors.Config{
		// AllowOrigins: []string{"http://localhost:3000"},
		// AllowMethods: []string{"POST", "GET"},
		AllowHeaders: []string{"Content-Type", "Authorization"},

		// 允许正式请求，响应带的header
		//ExposeHeaders: []string{"Content-Length"},

		// 是否允许带 cookie 之类的
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				// 你的开发环境
				return true
			}
			// 其他开发环境：你的域名
			return strings.Contains(origin, "qiuwenjuan.top")
		},
		MaxAge: 12 * time.Hour,
	}))

	// 方式1: 使用cookie
	// 登录校验模块: 初始化
	// 登录校验模块步骤1: 生成session, 放在cookie里
	//store := cookie.NewStore([]byte("secret"))

	// 方式2: 基于内存的
	// store := memstore.NewStore([]byte("k6CswdUm75WKcbM68UQUuxVsHSpTCwgK"), []byte("eF1`yQ9>yT1`tH1,sJ0.zD8;mZ9~nC6("))

	// 方式3: 基于redis 主流
	// docker redis 端口:6379 密码为空
	store, err := redis.NewStore(16, "tcp", "localhost:6379", "",
		[]byte("k6CswdUm75WKcbM68UQUuxVsHSpTCwgK"), []byte("eF1`yQ9>yT1`tH1,sJ0.zD8;mZ9~nC6("))

	if err != nil {
		panic(err)
	}

	// session存在context中, context中
	server.Use(sessions.Sessions("mysession", store)) // 设置 我的session的名字是 mysession 到cookie中,
	// session存在context中

	// 校验模块步骤3: 使用 验证session
	// v1
	server.Use(middleware.NewLoginMiddlewareBuilder().
		IgnorePaths("/users/signup").
		IgnorePaths("/users/login").Build()) //  链式调用

	return server
}

// 2. 初始化 DDD分层结构
func initUser(db *gorm.DB) *web.UserHandler {
	// DDD架构
	ud := dao2.NewUserDAO(db)
	repo := repository.NewUserRepository(ud)
	svc := service.NewUserService(repo)
	// 预编译 正则表达式（邮箱、 密码匹配） -- 优化项目性能, 提高校验速度
	u := web.NewUserHandler(svc)
	return u
}
