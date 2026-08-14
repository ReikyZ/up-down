package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhangrui/up-down/internal/config"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintln(os.Stderr, "Usage: up <file>")
		os.Exit(2)
	}
	if err := config.LoadDotEnv(".env"); err != nil {
		fatal(err)
	}
	serverURL := strings.TrimRight(os.Getenv("SERVER_URL"), "/")
	if serverURL == "" {
		fatal(fmt.Errorf("SERVER_URL is required in .env or environment"))
	}
	if _, err := url.ParseRequestURI(serverURL); err != nil {
		fatal(fmt.Errorf("invalid SERVER_URL: %w", err))
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fatal(err)
	}
	defer file.Close()
	reader, writerPipe := io.Pipe()
	writer := multipart.NewWriter(writerPipe)
	go func() {
		part, err := writer.CreateFormFile("file", filepath.Base(os.Args[1]))
		if err == nil {
			_, err = io.Copy(part, file)
		}
		if closeErr := writer.Close(); err == nil {
			err = closeErr
		}
		_ = writerPipe.CloseWithError(err)
	}()
	request, err := http.NewRequest(http.MethodPost, serverURL+"/upload", reader)
	if err != nil {
		fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(response.Body)
		fatal(fmt.Errorf("upload failed (%s): %s", response.Status, strings.TrimSpace(string(message))))
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		fatal(err)
	}
	fmt.Println(serverURL + result.URL)
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "up:", err); os.Exit(1) }
