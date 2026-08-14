package fileserver

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUploadAndDownload(t *testing.T) {
	server, err := New(Config{UploadDir: t.TempDir(), MinFreeBytes: 0, MaxUploadBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "example.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("file contents"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("upload status = %d: %s", response.Code, response.Body.String())
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	download := httptest.NewRecorder()
	server.Handler().ServeHTTP(download, httptest.NewRequest(http.MethodGet, result.URL, nil))
	contents, _ := io.ReadAll(download.Result().Body)
	if download.Code != http.StatusOK || string(contents) != "file contents" {
		t.Fatalf("download status/body = %d/%q", download.Code, contents)
	}
}

func TestOldestFileSelectsOldestUpload(t *testing.T) {
	dir := t.TempDir()
	oldest := filepath.Join(dir, "oldest.txt")
	newest := filepath.Join(dir, "newest.txt")
	for _, path := range []string{oldest, newest} {
		if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	oldTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(oldest, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	selected, err := oldestFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if selected != oldest {
		t.Fatalf("oldestFile() = %q, want %q", selected, oldest)
	}
}
