package handle

import (
	"fmt"
	"path"
	"strings"

	"gorm.io/driver/mysql"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"gorm.io/gorm"

	rPool "Clould/webook/internal/repository/cache/redis"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

// 上传文件初始化分片信息
type MultipartUploadInfo struct {
	FileHash   string
	FileSize   int64
	UploadId   string
	ChunkSize  int
	ChunkCount int
}

// 初始化分块上传
func InitialMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求参数
	r.ParseForm()
	userName := r.Form.Get("username")
	fileHash := r.Form.Get("filehash")
	fileSize, err := strconv.Atoi(r.Form.Get("filesize"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 2. 获得redis的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 生成分块上传的初始化信息
	upInfo := MultipartUploadInfo{
		FileHash:   fileHash,
		FileSize:   int64(fileSize),
		UploadId:   userName + fmt.Sprintf("%x", time.Now().Nanosecond()), // 文件id生成规则: 文件名 + 时间戳
		ChunkSize:  5 * 1024 * 1024,                                       //5M
		ChunkCount: int(math.Ceil((float64(fileSize)) / (5 * 1024 * 1024))),
	}
	// 4. 将初始化的的信息写入redis缓存
	rConn.Do("HSET", "MP"+upInfo.UploadId, "chunkcount", upInfo.ChunkCount)
	rConn.Do("HSET", "MP"+upInfo.UploadId, "chunkhash", upInfo.FileHash)
	rConn.Do("HSET", "MP"+upInfo.UploadId, "chunksize", upInfo.FileSize)

	// 5. 将响应初始化的数据返回到客户端
	w.WriteHeader(http.StatusOK)
}

// 上传文件分块
func UploadPartHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求参数
	r.ParseForm()
	//userName := r.Form.Get("username")
	uploadId := r.Form.Get("uploadid")
	chunkIndex := r.Form.Get("index")

	// 2. 获得redis连接池中的一个来连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 获取文件句柄, 用于存储文件分块内容
	fpath := "/data/" + uploadId + "/" + chunkIndex
	os.MkdirAll(path.Dir(fpath), 0744)
	fd, err := os.Create(fpath)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer fd.Close()
	buf := make([]byte, 1024*1024) // 每次读1M
	for {
		n, err := r.Body.Read(buf)
		fd.Write(buf[:n])
		if err != nil {
			break
		}
	}

	// 4. 更新redis缓存状态
	rConn.Do("HSET", "MP"+uploadId, "chkidx"+chunkIndex, 1)

	// 5. 返回处理结果到客户端
	w.WriteHeader(http.StatusOK)
}

// 通知上传合并接口
func CompleteMultipartUploadHandler(w http.ResponseWriter, r *http.Request, ctx *gin.Context) {
	// 1. 解析请求参数
	r.ParseForm()
	upId := r.Form.Get("uploadid")
	userName := r.Form.Get("username")
	fileHash := r.Form.Get("filehash")
	fileSize, err := strconv.Atoi(r.Form.Get("filesize"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	filename := r.Form.Get("filename")

	// 2. 获得redis连接池里面的连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 通过uploadid查询redis并判断是否所有分块上传完成
	data, err := rConn.Do("HSETALL", "MP"+upId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	totalCount := 0
	chunkCount := 0
	for i := 0; i < len(data); i += 2 {
		k := string(data[i].([]byte))
		v := string(data[i+1].([]byte))
		if k == "chunkcount" {
			totalCount, _ = strconv.Atoi(v)
		} else if strings.HasPrefix(k, "chkidx_") && v == "1" {
			chunkCount += 1
		}
	}
	if totalCount != chunkCount {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// 4. TODO: 合并分块

	// 5. 更新唯一文件表以及用户表
	FileUploadFinsh(ctx, userName, fileHash, filename, int64(fileSize))
	// 6. 响应处理结果
	w.WriteHeader(http.StatusOK)
}

func FileUploadFinsh(ctx *gin.Context, username string, filehash string, filename string, filesize int64) {
	db, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13316)/webook"))
	if err != nil {
		panic("无法连接到数据库")
	}
	now := time.Now().UnixMilli()
	session := sessions.Default(ctx)
	userID := session.Get("userId")
	userIDInt, err := strconv.ParseInt(userID.(string), 10, 64)
	if err != nil {
	}
	file := File{
		UserId:   userIDInt,
		Username: username,
		Filename: filename,
		Filehash: filehash,
		Filesize: filesize,
		Ctime:    now,
		Utime:    now,
	}
	err = db.Create(&file).Error
	if err != nil {
		return
	}
}

type File struct {
	Id       int64 `gorm:"primarykey, autoIncrement"`
	UserId   int64
	Username string
	Filename string
	Filehash string
	Filesize int64
	// 创建时间, 毫秒数
	Ctime int64
	// 更新时间, 毫秒数
	Utime int64
}
