package fileserver

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Config struct {
	UploadDir      string
	MinFreeBytes   uint64
	MaxUploadBytes int64
}

type Server struct {
	config  Config
	storage *storage
}

func New(config Config) (*Server, error) {
	if config.UploadDir == "" {
		return nil, errors.New("upload directory is required")
	}
	if config.MaxUploadBytes <= 0 {
		return nil, errors.New("maximum upload size must be positive")
	}
	storage, err := newStorage(config.UploadDir, config.MinFreeBytes)
	if err != nil {
		return nil, err
	}
	return &Server{config: config, storage: storage}, nil
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
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "invalid upload or file exceeds upload limit", http.StatusBadRequest)
		return
	}

	part, err := uploadPart(reader)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "file exceeds upload limit", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer part.Close()

	name, err := s.storage.Save(part, part.FileName())
	if err != nil {
		switch {
		case errors.Is(err, ErrUploadTooLarge):
			http.Error(w, "file exceeds upload limit", http.StatusRequestEntityTooLarge)
		case errors.Is(err, ErrInsufficientStorage):
			http.Error(w, "storage is full: "+err.Error(), http.StatusInsufficientStorage)
		default:
			http.Error(w, "could not save upload", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"filename": name, "url": "/files/" + name})
}
