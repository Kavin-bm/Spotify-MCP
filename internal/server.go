// Package internal wires together the Spotify provider, the service layer,
// and the MCP server (served over SSE or Streamable HTTP).
package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Kavin-bm/Spotify-MCP/config"
	spotifyhandler "github.com/Kavin-bm/Spotify-MCP/internal/spotify"
)

// Server runs the MCP HTTP server.
type Server struct {
	cfg     *config.Config
	svc     *Service
	httpSrv *http.Server
}

// NewServer creates a Server from the given config.
func NewServer(cfg *config.Config) (*Server, error) {
	handler, err := spotifyhandler.NewHandler(cfg.ClientID, cfg.ClientSecret, cfg.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create spotify handler: %w", err)
	}

	svc := NewService(handler, cfg)
	return &Server{cfg: cfg, svc: svc}, nil
}

// Start begins serving MCP over SSE (HTTP). Non-blocking.
func (s *Server) Start() error {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "spotify-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Instructions: buildMCPInstructions(s.cfg),
	})

	registerTools(mcpServer, s.svc)

	sseHandler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server {
		return mcpServer
	})

	mux := http.NewServeMux()
	mux.Handle("/", sseHandler)

	s.httpSrv = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: mux,
	}

	go func() {
		slog.Info("Spotify MCP server listening", "addr", s.httpSrv.Addr)
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("MCP server error", "err", err)
		}
	}()
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop() error {
	if s.httpSrv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5_000_000_000)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}
