package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type LocalEngine struct {
	root      string
	chunkRoot string
}

func NewLocalEngine(root string, chunkRoot string) *LocalEngine {
	_ = os.MkdirAll(root, 0755)
	_ = os.MkdirAll(chunkRoot, 0755)
	return &LocalEngine{
		root:      root,
		chunkRoot: chunkRoot,
	}
}

func (e *LocalEngine) Store(ctx context.Context, uid int64, filename string, r io.Reader, size int64) (string, error) {
	path, err := e.filePath(e.root, uid, filename)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err = writeFile(path, r, size); err != nil {
		return "", err
	}
	return path, nil
}

func (e *LocalEngine) StoreChunk(ctx context.Context, uid int64, identifier string, chunkNumber int, r io.Reader, size int64) (string, error) {
	id := safeToken(identifier)
	if id == "" {
		return "", fmt.Errorf("非法 identifier")
	}
	dir := filepath.Join(e.chunkRoot, fmt.Sprintf("%d", uid), id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d", chunkNumber))
	if err := writeFile(path, r, size); err != nil {
		return "", err
	}
	return path, nil
}

func (e *LocalEngine) Merge(ctx context.Context, uid int64, filename string, chunkPaths []string) (string, error) {
	dest, err := e.filePath(e.root, uid, filename)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return "", err
	}
	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer out.Close()
	for _, p := range chunkPaths {
		if err = appendFile(out, p); err != nil {
			_ = os.Remove(dest)
			return "", err
		}
	}
	_ = e.Delete(ctx, chunkPaths)
	return dest, nil
}

func (e *LocalEngine) Delete(ctx context.Context, realPaths []string) error {
	for _, p := range realPaths {
		if p == "" {
			continue
		}
		_ = os.Remove(p)
	}
	return nil
}

func (e *LocalEngine) Read(ctx context.Context, realPath string, w io.Writer) error {
	f, err := os.Open(realPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func (e *LocalEngine) filePath(root string, uid int64, filename string) (string, error) {
	name := sanitizeFilename(filename)
	if name == "" {
		return "", fmt.Errorf("非法文件名")
	}
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, fmt.Sprintf("%d", uid))
	return filepath.Join(dir, token+"_"+name), nil
}

func writeFile(path string, r io.Reader, size int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if size > 0 {
		_, err = io.CopyN(f, r, size)
		return err
	}
	_, err = io.Copy(f, r)
	return err
}

func appendFile(out *os.File, path string) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(out, in)
	return err
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "." || name == ".." || name == string(filepath.Separator) {
		return ""
	}
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return name
}

func safeToken(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func randomToken() (string, error) {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
