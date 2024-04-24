package middleware

import (
	"fmt"
	"net/http"

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
	return func(ctx *gin.Context) {
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
		id := sess.Get("userId")
		fmt.Printf("id: %d\n", id)
		// 没有获取到id
		if id == nil {
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