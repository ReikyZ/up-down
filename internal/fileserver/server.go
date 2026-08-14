package fileserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	UploadDir      string
	MinFreeBytes   uint64
	MaxUploadBytes int64
}

type Server struct{ config Config }

func New(config Config) (*Server, error) {
	if config.UploadDir == "" {
		return nil, errors.New("upload directory is required")
	}
	if config.MaxUploadBytes <= 0 {
		return nil, errors.New("maximum upload size must be positive")
	}
	if err := os.MkdirAll(config.UploadDir, 0755); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}
	return &Server{config: config}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /upload", s.upload)
	mux.Handle("GET /files/", http.StripPrefix("/files/", http.FileServer(http.Dir(s.config.UploadDir))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return mux
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid upload or file exceeds upload limit", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "multipart field 'file' is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if err := s.ensureFreeSpace(); err != nil {
		http.Error(w, "storage is full: "+err.Error(), http.StatusInsufficientStorage)
		return
	}
	name, err := newFilename(header)
	if err != nil {
		http.Error(w, "could not create file name", http.StatusInternalServerError)
		return
	}
	path := filepath.Join(s.config.UploadDir, name)
	destination, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		http.Error(w, "could not save upload", http.StatusInternalServerError)
		return
	}
	_, copyErr := io.Copy(destination, file)
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		http.Error(w, "could not save upload", http.StatusInternalServerError)
		return
	}
	if err := s.ensureFreeSpace(); err != nil {
		_ = os.Remove(path)
		http.Error(w, "storage is full after upload: "+err.Error(), http.StatusInsufficientStorage)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"filename": name, "url": "/files/" + name})
}

func newFilename(header *multipart.FileHeader) (string, error) {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	base := filepath.Base(header.Filename)
	ext := filepath.Ext(base)
	if len(ext) > 32 || strings.ContainsAny(ext, "/\\") {
		ext = ""
	}
	return time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(random) + ext, nil
}

func (s *Server) ensureFreeSpace() error {
	for {
		free, err := freeBytes(s.config.UploadDir)
		if err != nil {
			return err
		}
		if free >= s.config.MinFreeBytes {
			return nil
		}
		oldest, err := oldestFile(s.config.UploadDir)
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
		files = append(files, candidate{filepath.Join(dir, entry.Name()), info.ModTime()})
	}
	if len(files) == 0 {
		return "", nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modified.Before(files[j].modified) })
	return files[0].path, nil
}

func UintEnv(key string, fallback uint64) (uint64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseUint(value, 10, 64)
}
