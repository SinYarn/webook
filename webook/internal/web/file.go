package web

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/service"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// FileHandler 云盘 HTTP 入口。
// 分层：web -> service.FileService -> repository -> dao / storage.Engine
// 业务（秒传、分片、目录树）在 service；这里只做鉴权、参数和响应。
type FileHandler struct {
	svc *service.FileService
	// tickets 给 <video>/<img> 用：媒体标签带不上 Authorization，
	// 所以 /files/raw 走短期签名 ticket，不走登录 JWT。任意副本都能验。
	tickets *previewTickets
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc, tickets: newPreviewTickets()}
}

func (h *FileHandler) RegisterRoutes(server *gin.Engine) {
	fg := server.Group("/files")
	// GET /files?parentId=
	fg.GET("", h.List)
	// POST /files/folder
	// POST /files/folder  同名复用，返回 {Id, Filename}
	fg.POST("/folder", h.CreateFolder)
	// GET /files/breadcrumbs?id=
	fg.GET("/breadcrumbs", h.Breadcrumbs)
	// POST /files/delete
	fg.POST("/delete", h.Delete)
	// POST /files/upload  普通上传
	fg.POST("/upload", h.Upload)
	// GET /files/download?id=  附件下载，支持 Range
	fg.GET("/download", h.Download)
	// POST /files/preview-ticket  换 raw 用的 ticket（需 JWT）
	fg.POST("/preview-ticket", h.PreviewTicket)
	// GET /files/raw?id=&ticket=  在线预览，JWT 白名单，ticket 鉴权
	fg.GET("/raw", h.Raw)
	// POST /files/sec-upload  秒传
	fg.POST("/sec-upload", h.SecUpload)
	// POST /files/chunk-upload  上传一片
	fg.POST("/chunk-upload", h.ChunkUpload)
	// GET /files/chunk-upload?identifier=  已上传片号（断点续传）
	fg.GET("/chunk-upload", h.UploadedChunks)
	// POST /files/merge
	fg.POST("/merge", h.Merge)
}

func (h *FileHandler) List(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	parentId := queryInt64(ctx, "parentId", 0)
	list, err := h.svc.List(ctx, uid, parentId)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	vos := make([]fileVO, 0, len(list))
	for _, f := range list {
		vos = append(vos, toFileVO(f))
	}
	ctx.JSON(http.StatusOK, vos)
}

func (h *FileHandler) Breadcrumbs(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	id := queryInt64(ctx, "id", 0)
	list, err := h.svc.Breadcrumbs(ctx, uid, id)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	vos := make([]breadcrumbVO, 0, len(list))
	for _, b := range list {
		vos = append(vos, breadcrumbVO{Id: b.Id, Filename: b.Filename})
	}
	ctx.JSON(http.StatusOK, vos)
}

func (h *FileHandler) CreateFolder(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	type FolderReq struct {
		ParentId   int64  `json:"parentId"`
		FolderName string `json:"folderName"`
	}
	var req FolderReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	uf, err := h.svc.CreateFolder(ctx, uid, req.ParentId, req.FolderName)
	if err == service.ErrEmptyFolderName {
		ctx.String(http.StatusOK, "文件夹名不能为空")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.JSON(http.StatusOK, folderVO{Id: uf.Id, Filename: uf.Filename})
}

func (h *FileHandler) Delete(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	type DeleteReq struct {
		Ids []int64 `json:"ids"`
	}
	var req DeleteReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	if len(req.Ids) == 0 {
		ctx.String(http.StatusOK, "请选择文件")
		return
	}
	err := h.svc.Delete(ctx, uid, req.Ids)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "删除成功")
}

func (h *FileHandler) Upload(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	src, err := file.Open()
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	defer src.Close()
	parentId := formInt64(ctx, "parentId", 0)
	err = h.svc.Upload(ctx, uid, parentId, file.Filename, src, file.Size)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "上传成功")
}

func (h *FileHandler) Download(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	id := queryInt64(ctx, "id", 0)
	if id == 0 {
		ctx.String(http.StatusOK, "文件不存在")
		return
	}
	filename, path, err := h.svc.FileForDownload(ctx, uid, id)
	if err == service.ErrCannotDownloadFolder {
		ctx.String(http.StatusOK, "不能下载文件夹")
		return
	}
	if err == service.ErrFileNotFound {
		ctx.String(http.StatusOK, "文件不存在")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	// attachment：浏览器另存为；ServeContent 处理 Range / 206
	h.serveFile(ctx, filename, path, "attachment", "application/octet-stream")
}

func (h *FileHandler) PreviewTicket(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	type TicketReq struct {
		Id int64 `json:"id"`
	}
	var req TicketReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	if req.Id == 0 {
		ctx.String(http.StatusOK, "文件不存在")
		return
	}
	// 先确认当前用户对这个 user_file 有权访问，再发票
	_, _, err := h.svc.FileForDownload(ctx, uid, req.Id)
	if err == service.ErrCannotDownloadFolder {
		ctx.String(http.StatusOK, "不能预览文件夹")
		return
	}
	if err == service.ErrFileNotFound {
		ctx.String(http.StatusOK, "文件不存在")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ticket, err := h.tickets.Issue(uid, req.Id)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"ticket": ticket,
		"ttl":    int(previewTicketTTL.Seconds()),
	})
}

func (h *FileHandler) Raw(ctx *gin.Context) {
	id := queryInt64(ctx, "id", 0)
	uid, fileId, ok := h.tickets.Lookup(ctx.Query("ticket"))
	if !ok || id == 0 || fileId != id {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	filename, path, err := h.svc.FileForDownload(ctx, uid, id)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}
	// inline + 真实 MIME，视频靠 Range 拖进度
	h.serveFile(ctx, filename, path, "inline", fileMIME(filename))
}

// serveFile 用标准库 ServeContent：一次实现整文件和断点续传。
func (h *FileHandler) serveFile(ctx *gin.Context, filename string, path string, disposition string, contentType string) {
	f, err := os.Open(path)
	if err != nil {
		ctx.String(http.StatusOK, "文件不存在")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.Header("Accept-Ranges", "bytes")
	ctx.Header("Content-Type", contentType)
	ctx.Header("Content-Disposition", disposition+"; filename*=UTF-8''"+url.QueryEscape(filename))
	http.ServeContent(ctx.Writer, ctx.Request, filename, st.ModTime(), f)
}

func (h *FileHandler) SecUpload(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	type SecReq struct {
		ParentId   int64  `json:"parentId"`
		Filename   string `json:"filename"`
		Identifier string `json:"identifier"`
	}
	var req SecReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	err := h.svc.SecUpload(ctx, uid, req.ParentId, req.Filename, req.Identifier)
	if err == service.ErrSecUploadMiss {
		ctx.String(http.StatusOK, "文件不存在")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "秒传成功")
}

func (h *FileHandler) ChunkUpload(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	src, err := file.Open()
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	defer src.Close()
	identifier := ctx.PostForm("identifier")
	filename := ctx.PostForm("filename")
	chunkNumber := formInt(ctx, "chunkNumber", 0)
	totalChunks := formInt(ctx, "totalChunks", 0)
	flag, err := h.svc.ChunkUpload(ctx, uid, identifier, filename, chunkNumber, totalChunks, src, file.Size)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.JSON(http.StatusOK, chunkVO{MergeFlag: flag})
}

func (h *FileHandler) UploadedChunks(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	identifier := ctx.Query("identifier")
	nums, err := h.svc.UploadedChunks(ctx, uid, identifier)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.JSON(http.StatusOK, uploadedChunkVO{UploadedChunks: nums})
}

func (h *FileHandler) Merge(ctx *gin.Context) {
	uid, ok := h.uid(ctx)
	if !ok {
		return
	}
	type MergeReq struct {
		ParentId   int64  `json:"parentId"`
		Filename   string `json:"filename"`
		Identifier string `json:"identifier"`
		TotalSize  int64  `json:"totalSize"`
	}
	var req MergeReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	err := h.svc.Merge(ctx, uid, req.ParentId, req.Filename, req.Identifier, req.TotalSize)
	if err != nil {
		log.Printf("files merge uid=%d filename=%s: %v", uid, req.Filename, err)
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "上传成功")
}

func (h *FileHandler) uid(ctx *gin.Context) (int64, bool) {
	c, ok := ctx.Get("claims")
	if !ok {
		ctx.String(http.StatusOK, "系统错误")
		return 0, false
	}
	claims, ok := c.(*UserClaims)
	if !ok {
		ctx.String(http.StatusOK, "系统错误")
		return 0, false
	}
	return claims.Uid, true
}

type fileVO struct {
	Id           int64
	ParentId     int64
	Filename     string
	FolderFlag   int
	FileSizeDesc string
	Utime        int64
}

type folderVO struct {
	Id       int64
	Filename string
}

type breadcrumbVO struct {
	Id       int64
	Filename string
}

type chunkVO struct {
	MergeFlag int
}

type uploadedChunkVO struct {
	UploadedChunks []int
}

func toFileVO(f domain.UserFile) fileVO {
	return fileVO{
		Id:           f.Id,
		ParentId:     f.ParentId,
		Filename:     f.Filename,
		FolderFlag:   f.FolderFlag,
		FileSizeDesc: f.FileSizeDesc,
		Utime:        f.Utime,
	}
}

func queryInt64(ctx *gin.Context, key string, def int64) int64 {
	v := ctx.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func formInt64(ctx *gin.Context, key string, def int64) int64 {
	v := ctx.PostForm(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func formInt(ctx *gin.Context, key string, def int) int {
	return int(formInt64(ctx, key, int64(def)))
}

func fileMIME(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".txt", ".log":
		return "text/plain; charset=utf-8"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".ogv":
		return "video/ogg"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".m4a":
		return "audio/mp4"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".opus":
		return "audio/opus"
	default:
		return "application/octet-stream"
	}
}

// ---------- 预览 ticket：签名 JWT，任意副本都能验，视频 Range 可重复用直到过期 ----------

const previewTicketTTL = 15 * time.Minute

var previewTicketKey = []byte("preview-ticket-hs256-webook-7kN2pQ")

type previewClaims struct {
	jwt.RegisteredClaims
	Uid    int64
	FileId int64
}

type previewTickets struct{}

func newPreviewTickets() *previewTickets {
	return &previewTickets{}
}

func (s *previewTickets) Issue(uid int64, fileId int64) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, previewClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(previewTicketTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Uid:    uid,
		FileId: fileId,
	})
	return tok.SignedString(previewTicketKey)
}

func (s *previewTickets) Lookup(id string) (int64, int64, bool) {
	if id == "" {
		return 0, 0, false
	}
	claims := &previewClaims{}
	tok, err := jwt.ParseWithClaims(id, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrSignatureInvalid
		}
		return previewTicketKey, nil
	})
	if err != nil || tok == nil || !tok.Valid || claims.Uid == 0 || claims.FileId == 0 {
		return 0, 0, false
	}
	return claims.Uid, claims.FileId, true
}
