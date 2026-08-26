package storage

import (
	"context"
	"io"
)

// Engine 屏蔽本地磁盘 / 对象存储的区别
type Engine interface {
	Store(ctx context.Context, uid int64, filename string, r io.Reader, size int64) (realPath string, err error)
	StoreChunk(ctx context.Context, uid int64, identifier string, chunkNumber int, r io.Reader, size int64) (realPath string, err error)
	Merge(ctx context.Context, uid int64, filename string, chunkPaths []string) (realPath string, err error)
	Delete(ctx context.Context, realPaths []string) error
	Read(ctx context.Context, realPath string, w io.Writer) error
}
