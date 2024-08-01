package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/zde37/Hive/internal/config"
	"github.com/zde37/Hive/internal/handler"
	"github.com/zde37/Hive/internal/ipfs"
)

func main() {
	config := config.Load(os.Getenv("IPFS_RPC_ADDR"), os.Getenv("IPFS_WEB_UI_ADDR"),
		os.Getenv("IPFS_GATEWAY_ADDR"), os.Getenv("SERVER_ADDR"))

	rpc, err := ipfs.NewClient(config.RPC_ADDR)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := ipfs.NewClientImpl(rpc)
	hndl := handler.NewHandlerImpl(client)

	srv := &http.Server{
		Addr:    config.SERVER_ADDR,
		Handler: hndl.Mux(),
		// ReadTimeout:  30 * time.Second,
		// WriteTimeout: 30 * time.Second,
		// IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("server started on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("failed to start server: %v", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("shutting down server...")

	ctx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	srv.SetKeepAlivesEnabled(false)
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("could not gracefully shutdown the server: %v", err)
	}

	log.Println("server gracefully stopped")
}
