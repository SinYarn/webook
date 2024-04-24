package web

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/service"
	"fmt"
	"math"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc            *service.UserService
	emailRexExp    *regexp.Regexp
	passwordRexExp *regexp.Regexp
}

type FileHandler struct {
	svc *service.FileService
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

func NewFileHandler(svc *service.FileService) *FileHandler {

	return &FileHandler{
		svc: svc,
	}
}

func (h *UserHandler) RegisterRoutes(server *gin.Engine) {
	// REST 风格
	//server.POST("/user", h.SignUp)
	//server.PUT("/user", h.SignUp)
	//server.GET("/users/:username", h.Profile)
	ug := server.Group("/users")
	// POST /users/signup
	ug.POST("/signup", h.SignUp)
	// POST /users/login
	ug.POST("/login", h.Login)
	// POST /users/edit
	ug.POST("/edit", h.Edit)
	// GET /users/profile
	ug.GET("/profile", h.Profile)
}

func (f *FileHandler) FileRoutes(server *gin.Engine) {
	// REST 风格
	//server.POST("/user", h.SignUp)
	//server.PUT("/user", h.SignUp)
	//server.GET("/users/:username", h.Profile)
	ug := server.Group("/files")
	// POST /users/signup
	ug.POST("/upload", f.Upload)
	ug.GET("/download", f.Download)
	server.LoadHTMLFiles("list.html")
	ug.GET("/list", func(c *gin.Context) {
		// 渲染并返回 HTML 页面
		c.HTML(http.StatusOK, "list.html", nil)
	})
}

func (f *FileHandler) List(ctx *gin.Context) {
	//ctx.String(http.StatusOK, "LIST succeed")
	ctx.HTML(http.StatusOK, "list.html", nil)
}

func (u *FileHandler) Upload(ctx *gin.Context) {
	// 从请求中获取上传的文件

	file, err := ctx.FormFile("file")
	sess := sessions.Default(ctx)
	userId := sess.Get("userId")

	var uid int64
	if id, ok := userId.(int64); ok {
		fmt.Printf("id: %d\n", userId)
		uid = id
	} else {
		ctx.JSON(http.StatusOK, gin.H{"message": "用户session id错误"})
		return
	}

	if userId == nil {
		ctx.AbortWithStatus(http.StatusUnauthorized) // 返回 401(未授权) 错误码
		return
	}
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 将上传的文件保存到服务器
	dst := fmt.Sprintf("./uploads/%s", file.Filename)
	if err := ctx.SaveUploadedFile(file, dst); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
		return
	}

	err = u.svc.Upload(ctx, domain.File{
		UserId:     int64(uid),
		Username:   "1@qq.com",
		Filename:   file.Filename,
		Filesize:   int64(math.Ceil((float64(file.Size)) / (1024 * 1024))),
		UploadPath: dst,
		Filehash:   "o",
	})
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"message": "文件上传失败"})
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "文件上传成功"})
}

func (u *FileHandler) Download(ctx *gin.Context) {
	taskID := ctx.DefaultQuery("taskID", "0")
	filepath := "./uploads/" + taskID

	// 检查文件是否存在
	_, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 设置响应头，告诉浏览器这是一个要下载的文件
	ctx.Header("Content-Disposition", "attachment; filename="+taskID)
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.File(filepath)
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
	// 设置session的值, 放在session中
	sess.Set("userId", user.Id)
	sess.Save() // 保存 session 到 cookie中 (响应header中可以看到)

	ctx.String(http.StatusOK, "Login Succeed")
	return
}

func (u *UserHandler) Edit(ctx *gin.Context) {

}

func (u *UserHandler) Profile(ctx *gin.Context) {
	ctx.String(http.StatusOK, "这是你的Profile")
}
