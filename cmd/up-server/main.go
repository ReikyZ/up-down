package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhangrui/up-down/internal/config"
	"github.com/zhangrui/up-down/internal/fileserver"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatal(err)
	}
	minFree, err := config.UintEnv("MIN_FREE_BYTES", 1<<30)
	if err != nil {
		log.Fatalf("MIN_FREE_BYTES: %v", err)
	}
	maxUpload, err := config.UintEnv("MAX_UPLOAD_BYTES", 10<<30)
	if err != nil {
		log.Fatalf("MAX_UPLOAD_BYTES: %v", err)
	}
	server, err := fileserver.New(fileserver.Config{UploadDir: config.StringEnv("UPLOAD_DIR", "./uploads"), MinFreeBytes: minFree, MaxUploadBytes: int64(maxUpload)})
	if err != nil {
		log.Fatal(err)
	}
	address := config.StringEnv("LISTEN_ADDR", ":8080")
	fmt.Printf("upload server listening on %s\n", address)
	httpServer := &http.Server{
		Addr:              address,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("shutdown server: %v", err)
	}
}
