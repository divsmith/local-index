package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Server struct {
	config *config.Config
	server *http.Server
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			ReadTimeout:  time.Duration(cfg.Timeout) * time.Second,
			WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) SetupRoutes() {
	http.HandleFunc("/", s.handleHome)
	http.HandleFunc("/health", s.handleHealth)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}
