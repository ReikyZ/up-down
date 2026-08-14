package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zhangrui/up-down/internal/config"
	"github.com/zhangrui/up-down/internal/fileserver"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatal(err)
	}
	minFree, err := fileserver.UintEnv("MIN_FREE_BYTES", 1<<30)
	if err != nil {
		log.Fatalf("MIN_FREE_BYTES: %v", err)
	}
	maxUpload, err := fileserver.UintEnv("MAX_UPLOAD_BYTES", 10<<30)
	if err != nil {
		log.Fatalf("MAX_UPLOAD_BYTES: %v", err)
	}
	server, err := fileserver.New(fileserver.Config{UploadDir: env("UPLOAD_DIR", "./uploads"), MinFreeBytes: minFree, MaxUploadBytes: int64(maxUpload)})
	if err != nil {
		log.Fatal(err)
	}
	address := env("LISTEN_ADDR", ":8080")
	fmt.Printf("upload server listening on %s\n", address)
	log.Fatal(http.ListenAndServe(address, server.Handler()))
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
