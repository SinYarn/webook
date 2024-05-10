package web

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/service"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	jwt "github.com/golang-jwt/jwt/v5"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc            *service.UserService
	emailRexExp    *regexp.Regexp
	passwordRexExp *regexp.Regexp
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	const (
		// 正则表达式校验
		emailRegexPattern = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
		// 和上面比起来，用 ` 看起来就比较清爽
		passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
	)

	emailExp := regexp.MustCompile(emailRegexPattern, regexp.None)
	passwordExp := regexp.MustCompile(passwordRegexPattern, regexp.None)

	return &UserHandler{
		svc:            svc,
		emailRexExp:    emailExp,
		passwordRexExp: passwordExp,
	}
}

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {
	// REST 风格
	//server.POST("/user", uu.SignUp)
	//server.PUT("/user", uu.SignUp)
	//server.GET("/users/:username", uu.Profile)
	ug := server.Group("/users")
	// POST /users/signup
	ug.POST("/signup", u.SignUp)
	// POST /users/login
	ug.POST("/login", u.LoginJWT)
	// POST /users/edit
	ug.POST("/edit", u.Edit)
	// GET /users/profile
	ug.GET("/profile", u.ProfileJWT)
}

func (u *UserHandler) SignUp(ctx *gin.Context) {

	type SignUpReq struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	var req SignUpReq
	// bind 方法根据Content-Type 来解析请求数据到req
	// 解析错误， 返回400错误
	if err := ctx.Bind(&req); err != nil {
		return
	}

	ok, err := u.emailRexExp.MatchString(req.Email)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "非法邮箱格式")
		return
	}

	if req.ConfirmPassword != req.Password {
		ctx.String(http.StatusOK, "两次输入的密码不一致")
		return
	}

	ok, err = u.passwordRexExp.MatchString(req.Password)
	if err != nil {
		// 记录日志
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "密码必须大于8位， 包含数字、特殊字符")
		return
	}

	// 调用svc的方法
	err = u.svc.SignUp(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	})
	// 最佳实现 errors.Is(err, service.ErrUserDuplicateEmail)
	if err == service.ErrUserDuplicateEmail {
		ctx.String(http.StatusOK, "邮箱冲突")
		return
	}

	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "SignUp Succed")
	fmt.Printf("%v\n", req)
	return
}

func (u *UserHandler) Login(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// bind 获取登录的参数
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	user, err := u.svc.Login(ctx, req.Email, req.Password)
	if err == service.ErrInvalidUserOrPassword {
		ctx.String(http.StatusOK, "用户名或者密码不对")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 登录校验模块步骤2: 写入session的userID
	// 在这里登录成功了
	// 设置session 拿到session
	sess := sessions.Default(ctx)
	sess.Options(sessions.Options{
		MaxAge: 60, // 会话的最大存活时间, -1失效
	})
	// 设置session的值, 放在session中
	sess.Set("userId", user.Id)
	sess.Save() // 保存 session 到 cookie中 (响应header中可以看到)

	ctx.String(http.StatusOK, "Login Succeed")
	return
}

func (u *UserHandler) LoginJWT(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// bind 获取登录的参数
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	user, err := u.svc.Login(ctx, req.Email, req.Password)
	if err == service.ErrInvalidUserOrPassword {
		ctx.String(http.StatusOK, "用户名或者密码不对")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 登录校验模块步骤2:
	// 生成JWT token
	// TODO: 在jwt里添加个人数据
	// 自定义 claims结构体 里面放userId
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
		Uid:       user.Id,
		UserAgent: ctx.Request.UserAgent(),
	}

	// 生成token 同时把claims 放入
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString([]byte("k6CswdUm75WKcbM68UQUuxVsHSpTCwgK"))
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.Header("x-jwt-token", tokenStr)
	fmt.Println("tokenStr: ", tokenStr)
	fmt.Println(user)
	ctx.String(http.StatusOK, "Login Succeed, Use token")
	return
}

func (u *UserHandler) Logout(ctx *gin.Context) {
	sess := sessions.Default(ctx)
	sess.Options(sessions.Options{
		MaxAge: -1, // 会话的最大存活时间, -1失效
	})
	// 设置session的值, 放在session中
	// sess.Set("userId", user.Id)
	sess.Save() // 保存 session 到 cookie中 (响应header中可以看到)

	ctx.String(http.StatusOK, "Logout Succeed")
	return
}

func (u *UserHandler) Edit(ctx *gin.Context) {

}

func (u *UserHandler) ProfileJWT(ctx *gin.Context) {
	c, ok := ctx.Get("claims")
	// 必然有claims
	if !ok {
		// 监控住这里
		ctx.String(http.StatusOK, "系统错误")
	}
	// ok 代表是不是 *UserClaims 如果断言成功
	claims, ok := c.(*UserClaims)
	if !ok {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "你的 Profile")
	println("ProfileJWT - claims.Uid: ", claims.Uid)
}

func (u *UserHandler) Profile(ctx *gin.Context) {
	ctx.String(http.StatusOK, "这是你的Profile")
}

type UserClaims struct {
	jwt.RegisteredClaims
	// 声明自己要放进去token里面的数据
	// 可以自己随便加字段, 不要放敏感信息
	Uid       int64
	UserAgent string
}
