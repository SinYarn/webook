package domain

// 云盘领域模型。目录树（UserFile）和物理文件（File）分开：
// 同一份物理文件可被多个用户节点引用（秒传 / 去重）。

// UserFile 用户看到的目录树节点。ParentId=0 表示虚拟根，库里没有根行。
type UserFile struct {
	Id           int64
	UserId       int64
	ParentId     int64
	RealFileId   int64  // 指向 files.id；文件夹为 0
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

// File 落盘后的物理文件。Identifier 是内容 MD5，全局唯一，用来秒传。
type File struct {
	Id           int64
	Filename     string
	RealPath     string
	FileSize     int64
	FileSizeDesc string
	Identifier   string // 内容 MD5
	Ctime        int64
	Utime        int64
}

// FileChunk 未合并的分片。断点续传只问「哪些 ChunkNumber 已在」，不另建任务表。
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
