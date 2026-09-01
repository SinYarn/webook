package dao

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrFileNotFound            = gorm.ErrRecordNotFound
	ErrFileDuplicateIdentifier = errors.New("文件标识冲突")
)

// FileDAO 云盘三张表的 gorm 访问。只出现在 repository 里。
type FileDAO struct {
	db *gorm.DB
}

func NewFileDAO(db *gorm.DB) *FileDAO {
	return &FileDAO{db: db}
}

func (dao *FileDAO) InsertFile(ctx context.Context, f File) (File, error) {
	now := time.Now().UnixMilli()
	f.Ctime = now
	f.Utime = now
	err := dao.db.WithContext(ctx).Create(&f).Error
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		const uniqueConflictsErrNo uint16 = 1062
		if mysqlErr.Number == uniqueConflictsErrNo {
			return File{}, ErrFileDuplicateIdentifier
		}
	}
	return f, err
}

func (dao *FileDAO) FindFileByIdentifier(ctx context.Context, identifier string) (File, error) {
	var f File
	err := dao.db.WithContext(ctx).Where("identifier = ?", identifier).Limit(1).Find(&f).Error
	if err != nil {
		return File{}, err
	}
	if f.Id == 0 {
		return File{}, ErrFileNotFound
	}
	return f, nil
}

func (dao *FileDAO) FindFileById(ctx context.Context, id int64) (File, error) {
	var f File
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&f).Error
	return f, err
}

func (dao *FileDAO) DeleteFile(ctx context.Context, id int64) error {
	return dao.db.WithContext(ctx).Where("id = ?", id).Delete(&File{}).Error
}

func (dao *FileDAO) CountUserFilesByRealFileId(ctx context.Context, realFileId int64) (int64, error) {
	var n int64
	err := dao.db.WithContext(ctx).Model(&UserFile{}).
		Where("real_file_id = ?", realFileId).Count(&n).Error
	return n, err
}

func (dao *FileDAO) InsertUserFile(ctx context.Context, uf UserFile) (UserFile, error) {
	now := time.Now().UnixMilli()
	uf.Ctime = now
	uf.Utime = now
	err := dao.db.WithContext(ctx).Create(&uf).Error
	return uf, err
}

func (dao *FileDAO) FindUserFileById(ctx context.Context, uid int64, id int64) (UserFile, error) {
	var uf UserFile
	err := dao.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, uid).First(&uf).Error
	return uf, err
}

// FindUserFileByName 同一父目录下按名字+类型查找，给建目录去重用。
func (dao *FileDAO) FindUserFileByName(ctx context.Context, uid int64, parentId int64, name string, folderFlag int) (UserFile, error) {
	var uf UserFile
	err := dao.db.WithContext(ctx).
		Where("user_id = ? AND parent_id = ? AND filename = ? AND folder_flag = ?", uid, parentId, name, folderFlag).
		Limit(1).Find(&uf).Error
	if err != nil {
		return UserFile{}, err
	}
	if uf.Id == 0 {
		return UserFile{}, ErrFileNotFound
	}
	return uf, nil
}

func (dao *FileDAO) ListUserFiles(ctx context.Context, uid int64, parentId int64) ([]UserFile, error) {
	var list []UserFile
	err := dao.db.WithContext(ctx).
		Where("user_id = ? AND parent_id = ?", uid, parentId).
		Order("folder_flag desc, filename asc").
		Find(&list).Error
	return list, err
}

func (dao *FileDAO) ListChildren(ctx context.Context, uid int64, parentId int64) ([]UserFile, error) {
	return dao.ListUserFiles(ctx, uid, parentId)
}

func (dao *FileDAO) DeleteUserFile(ctx context.Context, uid int64, id int64) error {
	return dao.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, uid).
		Delete(&UserFile{}).Error
}

func (dao *FileDAO) InsertChunk(ctx context.Context, c FileChunk) error {
	now := time.Now().UnixMilli()
	c.Ctime = now
	c.Utime = now
	return dao.db.WithContext(ctx).Create(&c).Error
}

func (dao *FileDAO) FindChunk(ctx context.Context, uid int64, identifier string, chunkNumber int) (FileChunk, error) {
	var c FileChunk
	err := dao.db.WithContext(ctx).
		Where("create_user = ? AND identifier = ? AND chunk_number = ?", uid, identifier, chunkNumber).
		Limit(1).Find(&c).Error
	if err != nil {
		return FileChunk{}, err
	}
	if c.Id == 0 {
		return FileChunk{}, ErrFileNotFound
	}
	return c, nil
}

func (dao *FileDAO) ListChunks(ctx context.Context, uid int64, identifier string) ([]FileChunk, error) {
	var list []FileChunk
	now := time.Now().UnixMilli()
	err := dao.db.WithContext(ctx).
		Where("create_user = ? AND identifier = ? AND expiration_time > ?", uid, identifier, now).
		Order("chunk_number asc").
		Find(&list).Error
	return list, err
}

func (dao *FileDAO) DeleteChunks(ctx context.Context, uid int64, identifier string) error {
	return dao.db.WithContext(ctx).
		Where("create_user = ? AND identifier = ?", uid, identifier).
		Delete(&FileChunk{}).Error
}

// File 物理文件表 files。Identifier 唯一。
type File struct {
	Id           int64 `gorm:"primarykey, autoIncrement"`
	Filename     string
	RealPath     string
	FileSize     int64
	FileSizeDesc string
	Identifier   string `gorm:"unique"`
	Ctime        int64
	Utime        int64
}

func (File) TableName() string {
	return "files"
}

// UserFile 用户目录表 user_files。ParentId=0 为根下节点。
type UserFile struct {
	Id           int64 `gorm:"primarykey, autoIncrement"`
	UserId       int64 `gorm:"index:idx_user_parent"`
	ParentId     int64 `gorm:"index:idx_user_parent"`
	RealFileId   int64
	Filename     string
	FolderFlag   int
	FileSizeDesc string
	Ctime        int64
	Utime        int64
}

func (UserFile) TableName() string {
	return "user_files"
}

// FileChunk 分片表 file_chunks。expiration_time 过期后 List 不再返回。
type FileChunk struct {
	Id             int64 `gorm:"primarykey, autoIncrement"`
	Identifier     string
	RealPath       string
	ChunkNumber    int
	ExpirationTime int64
	CreateUser     int64
	Ctime          int64
	Utime          int64
}

func (FileChunk) TableName() string {
	return "file_chunks"
}
