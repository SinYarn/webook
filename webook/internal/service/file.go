package service

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/repository"
	"Clould/webook/internal/service/storage"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

var (
	ErrFileNotFound         = repository.ErrFileNotFound
	ErrSecUploadMiss        = errors.New("文件不存在")
	ErrNotFolder            = errors.New("父目录不存在")
	ErrCannotDownloadFolder = errors.New("不能下载文件夹")
	ErrEmptyFolderName      = errors.New("文件夹名不能为空")
)

// FileService 云盘用例：目录、上传、秒传、分片合并、删除回收。
// 不碰 HTTP；磁盘走 storage.Engine，表走 repository。
type FileService struct {
	repo   *repository.FileRepository
	engine storage.Engine
}

func NewFileService(repo *repository.FileRepository, engine storage.Engine) *FileService {
	return &FileService{
		repo:   repo,
		engine: engine,
	}
}

// List parentId=0 为虚拟根。
func (svc *FileService) List(ctx context.Context, uid int64, parentId int64) ([]domain.UserFile, error) {
	if err := svc.checkParent(ctx, uid, parentId); err != nil {
		return nil, err
	}
	return svc.repo.ListUserFiles(ctx, uid, parentId)
}

func (svc *FileService) Breadcrumbs(ctx context.Context, uid int64, id int64) ([]domain.Breadcrumb, error) {
	if id == 0 {
		return []domain.Breadcrumb{}, nil
	}
	var chain []domain.Breadcrumb
	cur := id
	for i := 0; i < 64 && cur != 0; i++ {
		uf, err := svc.repo.FindUserFile(ctx, uid, cur)
		if err != nil {
			return nil, err
		}
		chain = append(chain, domain.Breadcrumb{Id: uf.Id, Filename: uf.Filename})
		cur = uf.ParentId
	}
	// 根 -> 当前
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

// CreateFolder 同名目录复用已有节点，方便按路径建树。
func (svc *FileService) CreateFolder(ctx context.Context, uid int64, parentId int64, name string) (domain.UserFile, error) {
	if name == "" {
		return domain.UserFile{}, ErrEmptyFolderName
	}
	if err := svc.checkParent(ctx, uid, parentId); err != nil {
		return domain.UserFile{}, err
	}
	exist, err := svc.repo.FindFolder(ctx, uid, parentId, name)
	if err == nil {
		return exist, nil
	}
	if err != repository.ErrFileNotFound {
		return domain.UserFile{}, err
	}
	return svc.repo.CreateUserFile(ctx, domain.UserFile{
		UserId:     uid,
		ParentId:   parentId,
		RealFileId: 0,
		Filename:   name,
		FolderFlag: domain.FolderYes,
	})
}

// Upload 普通上传：落盘、算 MD5、复用已有 File、再挂一条 UserFile。
func (svc *FileService) Upload(ctx context.Context, uid int64, parentId int64, filename string, r io.Reader, size int64) error {
	if err := svc.checkParent(ctx, uid, parentId); err != nil {
		return err
	}
	h := md5.New()
	tee := io.TeeReader(r, h)
	path, err := svc.engine.Store(ctx, uid, filename, tee, size)
	if err != nil {
		return err
	}
	identifier := hex.EncodeToString(h.Sum(nil))
	f, err := svc.saveOrReuseFile(ctx, uid, filename, path, size, identifier)
	if err != nil {
		_ = svc.engine.Delete(ctx, []string{path})
		return err
	}
	_, err = svc.repo.CreateUserFile(ctx, domain.UserFile{
		UserId:       uid,
		ParentId:     parentId,
		RealFileId:   f.Id,
		Filename:     filename,
		FolderFlag:   domain.FolderNo,
		FileSizeDesc: f.FileSizeDesc,
	})
	return err
}

// SecUpload 秒传：identifier 已有物理文件则只插用户节点，不写盘。
func (svc *FileService) SecUpload(ctx context.Context, uid int64, parentId int64, filename string, identifier string) error {
	if identifier == "" {
		return ErrSecUploadMiss
	}
	if err := svc.checkParent(ctx, uid, parentId); err != nil {
		return err
	}
	f, err := svc.repo.FindFileByIdentifier(ctx, identifier)
	if err == repository.ErrFileNotFound {
		return ErrSecUploadMiss
	}
	if err != nil {
		return err
	}
	_, err = svc.repo.CreateUserFile(ctx, domain.UserFile{
		UserId:       uid,
		ParentId:     parentId,
		RealFileId:   f.Id,
		Filename:     filename,
		FolderFlag:   domain.FolderNo,
		FileSizeDesc: f.FileSizeDesc,
	})
	return err
}

// ChunkUpload 保存一片。已存在的片直接跳过（断点续传）。
func (svc *FileService) ChunkUpload(ctx context.Context, uid int64, identifier string, filename string,
	chunkNumber int, totalChunks int, r io.Reader, size int64) (int, error) {
	if identifier == "" || chunkNumber <= 0 || totalChunks <= 0 {
		return 0, errors.New("参数错误")
	}
	existing, err := svc.repo.FindChunk(ctx, uid, identifier, chunkNumber)
	if err == nil && existing.Id > 0 {
		return svc.mergeFlag(ctx, uid, identifier, totalChunks)
	}
	if err != nil && err != repository.ErrFileNotFound {
		return 0, err
	}
	path, err := svc.engine.StoreChunk(ctx, uid, identifier, chunkNumber, r, size)
	if err != nil {
		return 0, err
	}
	err = svc.repo.CreateChunk(ctx, domain.FileChunk{
		Identifier:     identifier,
		RealPath:       path,
		ChunkNumber:    chunkNumber,
		ExpirationTime: time.Now().Add(24 * time.Hour).UnixMilli(),
		CreateUser:     uid,
	})
	if err != nil {
		_ = svc.engine.Delete(ctx, []string{path})
		return 0, err
	}
	return svc.mergeFlag(ctx, uid, identifier, totalChunks)
}

// UploadedChunks 返回已落盘的片号，前端用来跳过。
func (svc *FileService) UploadedChunks(ctx context.Context, uid int64, identifier string) ([]int, error) {
	list, err := svc.repo.ListChunks(ctx, uid, identifier)
	if err != nil {
		return nil, err
	}
	nums := make([]int, 0, len(list))
	for _, c := range list {
		nums = append(nums, c.ChunkNumber)
	}
	return nums, nil
}

// Merge 按片号 1..n 拼文件，删分片记录，再走秒传复用。
func (svc *FileService) Merge(ctx context.Context, uid int64, parentId int64, filename string, identifier string, totalSize int64) error {
	if err := svc.checkParent(ctx, uid, parentId); err != nil {
		return err
	}
	chunks, err := svc.repo.ListChunks(ctx, uid, identifier)
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return errors.New("该文件未找到分片记录")
	}
	paths := make([]string, 0, len(chunks))
	for i, c := range chunks {
		if c.ChunkNumber != i+1 {
			return errors.New("分片不完整")
		}
		paths = append(paths, c.RealPath)
	}
	path, err := svc.engine.Merge(ctx, uid, filename, paths)
	if err != nil {
		return err
	}
	_ = svc.repo.DeleteChunks(ctx, uid, identifier)
	f, err := svc.saveOrReuseFile(ctx, uid, filename, path, totalSize, identifier)
	if err != nil {
		_ = svc.engine.Delete(ctx, []string{path})
		return err
	}
	_, err = svc.repo.CreateUserFile(ctx, domain.UserFile{
		UserId:       uid,
		ParentId:     parentId,
		RealFileId:   f.Id,
		Filename:     filename,
		FolderFlag:   domain.FolderNo,
		FileSizeDesc: f.FileSizeDesc,
	})
	return err
}

// FileForDownload 校验归属后返回展示名和物理路径。web 层负责 Range / MIME。
func (svc *FileService) FileForDownload(ctx context.Context, uid int64, id int64) (string, string, error) {
	uf, err := svc.repo.FindUserFile(ctx, uid, id)
	if err != nil {
		return "", "", err
	}
	if uf.FolderFlag == domain.FolderYes {
		return "", "", ErrCannotDownloadFolder
	}
	f, err := svc.repo.FindFileById(ctx, uf.RealFileId)
	if err != nil {
		return "", "", err
	}
	return uf.Filename, f.RealPath, nil
}

func (svc *FileService) CopyFile(ctx context.Context, realPath string, w io.Writer) error {
	return svc.engine.Read(ctx, realPath, w)
}

// Delete 递归删目录树；物理文件引用计数为 0 才 gc。
func (svc *FileService) Delete(ctx context.Context, uid int64, ids []int64) error {
	for _, id := range ids {
		nodes, err := svc.collect(ctx, uid, id)
		if err != nil {
			return err
		}
		// 先删目录树，再按引用计数回收物理文件
		realIds := make([]int64, 0)
		for i := len(nodes) - 1; i >= 0; i-- {
			n := nodes[i]
			if n.FolderFlag == domain.FolderNo && n.RealFileId > 0 {
				realIds = append(realIds, n.RealFileId)
			}
			if err = svc.repo.DeleteUserFile(ctx, uid, n.Id); err != nil {
				return err
			}
		}
		for _, rid := range realIds {
			if err = svc.gcFile(ctx, rid); err != nil {
				return err
			}
		}
	}
	return nil
}

func (svc *FileService) collect(ctx context.Context, uid int64, id int64) ([]domain.UserFile, error) {
	uf, err := svc.repo.FindUserFile(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	res := []domain.UserFile{uf}
	if uf.FolderFlag != domain.FolderYes {
		return res, nil
	}
	children, err := svc.repo.ListChildren(ctx, uid, uf.Id)
	if err != nil {
		return nil, err
	}
	for _, c := range children {
		sub, err := svc.collect(ctx, uid, c.Id)
		if err != nil {
			return nil, err
		}
		res = append(res, sub...)
	}
	return res, nil
}

// gcFile 没有 user_files 再引用时删盘 + 删 files 行。
func (svc *FileService) gcFile(ctx context.Context, realFileId int64) error {
	n, err := svc.repo.CountByRealFileId(ctx, realFileId)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	f, err := svc.repo.FindFileById(ctx, realFileId)
	if err == repository.ErrFileNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	_ = svc.engine.Delete(ctx, []string{f.RealPath})
	return svc.repo.DeleteFile(ctx, f.Id)
}

// saveOrReuseFile 按 MD5 复用已有 File，避免重复占盘。
func (svc *FileService) saveOrReuseFile(ctx context.Context, _ int64, filename string, path string, size int64, identifier string) (domain.File, error) {
	exist, err := svc.repo.FindFileByIdentifier(ctx, identifier)
	if err == nil {
		_ = svc.engine.Delete(ctx, []string{path})
		return exist, nil
	}
	if err != repository.ErrFileNotFound {
		return domain.File{}, err
	}
	f, err := svc.repo.CreateFile(ctx, domain.File{
		Filename:     filename,
		RealPath:     path,
		FileSize:     size,
		FileSizeDesc: fileSizeDesc(size),
		Identifier:   identifier,
	})
	if err == repository.ErrFileDuplicateIdentifier {
		_ = svc.engine.Delete(ctx, []string{path})
		return svc.repo.FindFileByIdentifier(ctx, identifier)
	}
	return f, err
}

func (svc *FileService) mergeFlag(ctx context.Context, uid int64, identifier string, totalChunks int) (int, error) {
	list, err := svc.repo.ListChunks(ctx, uid, identifier)
	if err != nil {
		return 0, err
	}
	if len(list) >= totalChunks {
		return 1, nil
	}
	return 0, nil
}

func (svc *FileService) checkParent(ctx context.Context, uid int64, parentId int64) error {
	if parentId == 0 {
		return nil
	}
	uf, err := svc.repo.FindUserFile(ctx, uid, parentId)
	if err != nil {
		return ErrNotFolder
	}
	if uf.FolderFlag != domain.FolderYes {
		return ErrNotFolder
	}
	return nil
}

func fileSizeDesc(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
}
