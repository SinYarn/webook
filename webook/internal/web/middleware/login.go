package middleware

import (
	"encoding/gob"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Bulid 字段方便扩展
type LoginMiddlewareBuilder struct {
	paths []string
}

func NewLoginMiddlewareBuilder() *LoginMiddlewareBuilder {
	return &LoginMiddlewareBuilder{}
}

func (l *LoginMiddlewareBuilder) IgnorePaths(path string) *LoginMiddlewareBuilder {
	l.paths = append(l.paths, path)
	return l
}

func (l *LoginMiddlewareBuilder) Build() gin.HandlerFunc {
	gob.Register(time.Now())
	return func(ctx *gin.Context) {
		// 用Go的方式编码解码

		// signup 和 login 不需要校验
		for _, path := range l.paths {
			if ctx.Request.URL.Path == path {
				return
			}

		}
		sess := sessions.Default(ctx)
		/*
			下面一步包含了 冗余
			if sess == nil {
			// 没有登录
			ctx.AbortWithStatus(http.StatusUnauthorized) // 返回 401(未授权) 错误码
			return
		}*/

		// signup 和 login 不需要校验
		/*
			if ctx.Request.URL.Path == "/users/login" ||
				ctx.Request.URL.Path == "/users/signup" {
				return
			}
		*/
		userId := sess.Get("userId")
		if userId == nil {
			// 中断，不要往后执行，也就是不要执行后面的业务逻辑
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		updateTime := sess.Get("update_time")
		sess.Set("userId", userId)
		sess.Options(sessions.Options{
			MaxAge: 60,
		})
		now := time.Now()

		// 还没有刷新过, 第一次登录
		if updateTime == nil {
			sess.Set("update_time", now)
			if err := sess.Save(); err != nil {
				panic(err)
			}
			return
		}

		// uapdateTime 上次的时间
		updateTimeVal, _ := updateTime.(time.Time)
		// 10 * 1000 (ms) 刷新时间
		if now.Sub(updateTimeVal) > time.Second*10 {
			sess.Set("update_time", now)
			if err := sess.Save(); err != nil {
				panic(err)
			}
			return
		}
		sess.Set("update_time", now)
		// 没有获取到id
		if userId == nil {
			// 没有登录
			ctx.AbortWithStatus(http.StatusUnauthorized) // 返回 401(未授权) 错误码
			return
		}
	}
}

// 用户校验v2
func CheckLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sess := sessions.Default(ctx)
		if ctx.Request.URL.Path == "/users/login" ||
			ctx.Request.URL.Path == "/users/signup" {
			return
		}
		id := sess.Get("userId")
		// 没有获取到id
		if id == nil {
			// 没有登录
			ctx.AbortWithStatus(http.StatusUnauthorized) // 返回 401(未授权) 错误码
			return
		}
	}
}
