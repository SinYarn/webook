package middleware

import (
	"Clould/webook/internal/web"
	"encoding/gob"
	"log"
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
		// 拿到传入的claims
		claims := &web.UserClaims{}
		// ParseWithClaims 要传入指针
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("k6CswdUm75WKcbM68UQUuxVsHSpTCwgK"), nil
		})
		if err != nil {
			// 没登陆
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		/*		claims.ExpiresAt.Time.Before(time.Now()) {
				// 过期了
			}*/
		// err为nil, token 不为 nil
		if token == nil || !token.Valid || claims.Uid == 0 {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if claims.UserAgent != ctx.Request.UserAgent() {
			// 严重的安全问题
			// 要加监控
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 重新生成 token jwt 续约
		now := time.Now()
		// 每过十秒种刷新一次
		if claims.ExpiresAt.Sub(now) < time.Second*50 {
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
			tokenStr, err = token.SignedString([]byte("k6CswdUm75WKcbM68UQUuxVsHSpTCwgK"))
			if err != nil {
				// 记录日志
				log.Println("jwt 续约失败", err)
			}
			ctx.Header("x-jwt-token", tokenStr)
		}

		// 添加 到context中
		ctx.Set("claims", claims)
	}
}
