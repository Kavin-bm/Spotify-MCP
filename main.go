package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"

	"github.com/Kavin-bm/Spotify-MCP/config"
	"github.com/Kavin-bm/Spotify-MCP/internal"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	server, err := internal.NewServer(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	fmt.Printf("Spotify MCP server started on port %d\n", cfg.Port)
	fmt.Println("Press Ctrl+C to shutdown...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down gracefully...")
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	fmt.Println("Server stopped.")
}
