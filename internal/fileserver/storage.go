package fileserver

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	ErrInsufficientStorage = errors.New("insufficient storage")
	ErrUploadTooLarge      = errors.New("upload too large")
)

type storage struct {
	dir          string
	minFreeBytes uint64
	mu           sync.Mutex
}

func newStorage(dir string, minFreeBytes uint64) (*storage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}
	return &storage{dir: dir, minFreeBytes: minFreeBytes}, nil
}

func (s *storage) Save(source io.Reader, originalName string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureFreeSpace(); err != nil {
		return "", fmt.Errorf("prepare storage: %w: %w", ErrInsufficientStorage, err)
	}
	name, err := newFilename(originalName)
	if err != nil {
		return "", fmt.Errorf("generate file name: %w", err)
	}
	path := filepath.Join(s.dir, name)
	destination, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return "", fmt.Errorf("create destination: %w", err)
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		var maxBytesErr *http.MaxBytesError
		if errors.As(copyErr, &maxBytesErr) {
			return "", fmt.Errorf("write upload: %w", ErrUploadTooLarge)
		}
		return "", errors.New("write upload")
	}
	if err := s.ensureFreeSpace(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("reclaim storage after upload: %w: %w", ErrInsufficientStorage, err)
	}
	return name, nil
}

func uploadPart(reader *multipart.Reader) (*multipart.Part, error) {
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("multipart field 'file' is required")
		}
		if err != nil {
			return nil, fmt.Errorf("invalid multipart upload: %w", err)
		}
		if part.FormName() == "file" && part.FileName() != "" {
			return part, nil
		}
		part.Close()
	}
}

func newFilename(originalName string) (string, error) {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	ext := filepath.Ext(filepath.Base(originalName))
	if len(ext) > 32 || strings.ContainsAny(ext, "/\\") {
		ext = ""
	}
	return time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(random) + ext, nil
}

func (s *storage) ensureFreeSpace() error {
	for {
		free, err := freeBytes(s.dir)
		if err != nil {
			return err
		}
		if free >= s.minFreeBytes {
			return nil
		}
		oldest, err := oldestFile(s.dir)
		if err != nil {
			return err
		}
		if oldest == "" {
			return errors.New("no uploaded files can be deleted")
		}
		if err := os.Remove(oldest); err != nil {
			return fmt.Errorf("delete oldest upload: %w", err)
		}
	}
}

func freeBytes(path string) (uint64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, err
	}
	return stats.Bavail * uint64(stats.Bsize), nil
}

func oldestFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	type candidate struct {
		path     string
		modified time.Time
	}
	files := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		files = append(files, candidate{path: filepath.Join(dir, entry.Name()), modified: info.ModTime()})
	}
	if len(files) == 0 {
		return "", nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modified.Before(files[j].modified) })
	return files[0].path, nil
}
