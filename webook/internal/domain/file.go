package domain

// UserFile 用户目录树节点
type UserFile struct {
	Id           int64
	UserId       int64
	ParentId     int64
	RealFileId   int64
	Filename     string
	FolderFlag   int
	FileSizeDesc string
	Ctime        int64
	Utime        int64
}

const (
	FolderNo  = 0
	FolderYes = 1
)

// File 物理文件
type File struct {
	Id           int64
	Filename     string
	RealPath     string
	FileSize     int64
	FileSizeDesc string
	Identifier   string
	Ctime        int64
	Utime        int64
}

// FileChunk 分片记录
type FileChunk struct {
	Id             int64
	Identifier     string
	RealPath       string
	ChunkNumber    int
	ExpirationTime int64
	CreateUser     int64
	Ctime          int64
	Utime          int64
}

type Breadcrumb struct {
	Id       int64
	Filename string
}
