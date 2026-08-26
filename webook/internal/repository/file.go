package repository

import (
	"Clould/webook/internal/domain"
	"Clould/webook/internal/repository/dao"
	"context"
)

var (
	ErrFileNotFound            = dao.ErrFileNotFound
	ErrFileDuplicateIdentifier = dao.ErrFileDuplicateIdentifier
)

type FileRepository struct {
	dao *dao.FileDAO
}

func NewFileRepository(d *dao.FileDAO) *FileRepository {
	return &FileRepository{dao: d}
}

func (r *FileRepository) CreateFile(ctx context.Context, f domain.File) (domain.File, error) {
	po, err := r.dao.InsertFile(ctx, dao.File{
		Filename:     f.Filename,
		RealPath:     f.RealPath,
		FileSize:     f.FileSize,
		FileSizeDesc: f.FileSizeDesc,
		Identifier:   f.Identifier,
	})
	if err != nil {
		return domain.File{}, err
	}
	return toDomainFile(po), nil
}

func (r *FileRepository) FindFileByIdentifier(ctx context.Context, identifier string) (domain.File, error) {
	po, err := r.dao.FindFileByIdentifier(ctx, identifier)
	if err != nil {
		return domain.File{}, err
	}
	return toDomainFile(po), nil
}

func (r *FileRepository) FindFileById(ctx context.Context, id int64) (domain.File, error) {
	po, err := r.dao.FindFileById(ctx, id)
	if err != nil {
		return domain.File{}, err
	}
	return toDomainFile(po), nil
}

func (r *FileRepository) DeleteFile(ctx context.Context, id int64) error {
	return r.dao.DeleteFile(ctx, id)
}

func (r *FileRepository) CountByRealFileId(ctx context.Context, realFileId int64) (int64, error) {
	return r.dao.CountUserFilesByRealFileId(ctx, realFileId)
}

func (r *FileRepository) CreateUserFile(ctx context.Context, uf domain.UserFile) (domain.UserFile, error) {
	po, err := r.dao.InsertUserFile(ctx, dao.UserFile{
		UserId:       uf.UserId,
		ParentId:     uf.ParentId,
		RealFileId:   uf.RealFileId,
		Filename:     uf.Filename,
		FolderFlag:   uf.FolderFlag,
		FileSizeDesc: uf.FileSizeDesc,
	})
	if err != nil {
		return domain.UserFile{}, err
	}
	return toDomainUserFile(po), nil
}

func (r *FileRepository) FindUserFile(ctx context.Context, uid int64, id int64) (domain.UserFile, error) {
	po, err := r.dao.FindUserFileById(ctx, uid, id)
	if err != nil {
		return domain.UserFile{}, err
	}
	return toDomainUserFile(po), nil
}

func (r *FileRepository) ListUserFiles(ctx context.Context, uid int64, parentId int64) ([]domain.UserFile, error) {
	list, err := r.dao.ListUserFiles(ctx, uid, parentId)
	if err != nil {
		return nil, err
	}
	res := make([]domain.UserFile, 0, len(list))
	for _, po := range list {
		res = append(res, toDomainUserFile(po))
	}
	return res, nil
}

func (r *FileRepository) ListChildren(ctx context.Context, uid int64, parentId int64) ([]domain.UserFile, error) {
	return r.ListUserFiles(ctx, uid, parentId)
}

func (r *FileRepository) DeleteUserFile(ctx context.Context, uid int64, id int64) error {
	return r.dao.DeleteUserFile(ctx, uid, id)
}

func (r *FileRepository) CreateChunk(ctx context.Context, c domain.FileChunk) error {
	return r.dao.InsertChunk(ctx, dao.FileChunk{
		Identifier:     c.Identifier,
		RealPath:       c.RealPath,
		ChunkNumber:    c.ChunkNumber,
		ExpirationTime: c.ExpirationTime,
		CreateUser:     c.CreateUser,
	})
}

func (r *FileRepository) FindChunk(ctx context.Context, uid int64, identifier string, chunkNumber int) (domain.FileChunk, error) {
	po, err := r.dao.FindChunk(ctx, uid, identifier, chunkNumber)
	if err != nil {
		return domain.FileChunk{}, err
	}
	return toDomainChunk(po), nil
}

func (r *FileRepository) ListChunks(ctx context.Context, uid int64, identifier string) ([]domain.FileChunk, error) {
	list, err := r.dao.ListChunks(ctx, uid, identifier)
	if err != nil {
		return nil, err
	}
	res := make([]domain.FileChunk, 0, len(list))
	for _, po := range list {
		res = append(res, toDomainChunk(po))
	}
	return res, nil
}

func (r *FileRepository) DeleteChunks(ctx context.Context, uid int64, identifier string) error {
	return r.dao.DeleteChunks(ctx, uid, identifier)
}

func toDomainFile(po dao.File) domain.File {
	return domain.File{
		Id:           po.Id,
		Filename:     po.Filename,
		RealPath:     po.RealPath,
		FileSize:     po.FileSize,
		FileSizeDesc: po.FileSizeDesc,
		Identifier:   po.Identifier,
		Ctime:        po.Ctime,
		Utime:        po.Utime,
	}
}

func toDomainUserFile(po dao.UserFile) domain.UserFile {
	return domain.UserFile{
		Id:           po.Id,
		UserId:       po.UserId,
		ParentId:     po.ParentId,
		RealFileId:   po.RealFileId,
		Filename:     po.Filename,
		FolderFlag:   po.FolderFlag,
		FileSizeDesc: po.FileSizeDesc,
		Ctime:        po.Ctime,
		Utime:        po.Utime,
	}
}

func toDomainChunk(po dao.FileChunk) domain.FileChunk {
	return domain.FileChunk{
		Id:             po.Id,
		Identifier:     po.Identifier,
		RealPath:       po.RealPath,
		ChunkNumber:    po.ChunkNumber,
		ExpirationTime: po.ExpirationTime,
		CreateUser:     po.CreateUser,
		Ctime:          po.Ctime,
		Utime:          po.Utime,
	}
}
