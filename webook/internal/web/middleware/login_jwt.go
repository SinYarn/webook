package middleware

import (
	"encoding/gob"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
)

// LoginJWTMiddlewareBuilder 字段方便扩展
type LoginJWTMiddlewareBuilder struct {
	paths []string
}

func NewLoginJWTMiddlewareBuilder() *LoginJWTMiddlewareBuilder {
	return &LoginJWTMiddlewareBuilder{}
}

func (l *LoginJWTMiddlewareBuilder) IgnorePaths(path string) *LoginJWTMiddlewareBuilder {
	l.paths = append(l.paths, path)
	return l
}

func (l *LoginJWTMiddlewareBuilder) Build() gin.HandlerFunc {
	// 用Go的方式编码解码
	gob.Register(time.Now())
	return func(ctx *gin.Context) {
		// signup 和 login 不需要校验
		for _, path := range l.paths {
			if ctx.Request.URL.Path == path {
				return
			}
		}
		// 使用JWT 校验 jwt放在header里面
		tokenHeader := ctx.GetHeader("Authorization")
		if tokenHeader == "" {
			// 没登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		segs := strings.Split(tokenHeader, " ")
		if len(segs) != 2 {
			// 没登陆, 判断是不是2段
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		tokenStr := segs[1]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte("k6CswdUm75WKcbM68UQUuxVsHSpTCwgK"), nil
		})
		if err != nil {
			// 没登陆
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// err为nil, token 不为 nil
		if !token.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}
}
