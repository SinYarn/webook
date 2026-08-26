package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalEngineStoreReadDelete(t *testing.T) {
	dir := t.TempDir()
	e := NewLocalEngine(filepath.Join(dir, "files"), filepath.Join(dir, "chunks"))
	ctx := context.Background()
	content := []byte("hello webook")
	path, err := e.Store(ctx, 1, "a.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, string(filepath.Separator)+"1"+string(filepath.Separator)) {
		t.Fatalf("path should be under uid dir: %s", path)
	}
	var buf bytes.Buffer
	if err = e.Read(ctx, path, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != string(content) {
		t.Fatalf("got %s", buf.String())
	}
	if err = e.Delete(ctx, []string{path}); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file should be deleted")
	}
}

func TestLocalEngineChunkMerge(t *testing.T) {
	dir := t.TempDir()
	e := NewLocalEngine(filepath.Join(dir, "files"), filepath.Join(dir, "chunks"))
	ctx := context.Background()
	p1, err := e.StoreChunk(ctx, 2, "abc123", 1, strings.NewReader("hel"), 3)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := e.StoreChunk(ctx, 2, "abc123", 2, strings.NewReader("lo"), 2)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := e.Merge(ctx, 2, "b.txt", []string{p1, p2})
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(mustOpen(t, merged))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("got %s", data)
	}
}

func TestSanitizeRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	e := NewLocalEngine(filepath.Join(dir, "files"), filepath.Join(dir, "chunks"))
	path, err := e.Store(context.Background(), 1, "../etc/passwd", strings.NewReader("x"), 1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(path, "..") {
		t.Fatalf("path traversal leaked: %s", path)
	}
	if filepath.Base(path) == "passwd" || strings.HasSuffix(path, "passwd") {
		// basename is passwd after sanitize, that's ok as long as it's under uid dir
	}
	if !strings.Contains(filepath.Dir(path), filepath.Join(dir, "files", "1")) &&
		!strings.Contains(filepath.Dir(path), filepath.Join("files", "1")) {
		t.Fatalf("should stay under files/1: %s", path)
	}
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}
