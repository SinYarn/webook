package web

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/service"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	svc *service.FileService
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

func (h *FileHandler) RegisterRoutes(server *gin.Engine) {
	fg := server.Group("/files")
	fg.GET("", h.List)
	fg.POST("/folder", h.CreateFolder)
	fg.GET("/breadcrumbs", h.Breadcrumbs)
	fg.POST("/delete", h.Delete)
	fg.POST("/upload", h.Upload)
	fg.GET("/download", h.Download)
	fg.POST("/sec-upload", h.SecUpload)
	fg.POST("/chunk-upload", h.ChunkUpload)
	fg.GET("/chunk-upload", h.UploadedChunks)
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
	err := h.svc.CreateFolder(ctx, uid, req.ParentId, req.FolderName)
	if err == service.ErrEmptyFolderName {
		ctx.String(http.StatusOK, "文件夹名不能为空")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "创建成功")
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
	ctx.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(filename))
	ctx.Header("Content-Type", "application/octet-stream")
	err = h.svc.CopyFile(ctx, path, ctx.Writer)
	if err != nil {
		return
	}
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
