package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Server provides an HTTP server for serving Prometheus metrics
type Server struct {
	server   *http.Server
	logger   *zap.Logger
	registry *Registry
}

// ServerConfig holds configuration for the metrics server
type ServerConfig struct {
	Port    int           `env:"METRICS_PORT" envDefault:"9090"`
	Timeout time.Duration `env:"METRICS_TIMEOUT" envDefault:"30s"`
}

// NewServer creates a new metrics server instance
func NewServer(config ServerConfig, registry *Registry, logger *zap.Logger) *Server {
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	mux.Handle("/metrics", registry.Handler())

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"pub-metrics"}`))
	})

	// Ready check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","service":"pub-metrics"}`))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
		IdleTimeout:  config.Timeout * 2,
	}

	return &Server{
		server:   server,
		logger:   logger.Named("metrics-server"),
		registry: registry,
	}
}

// Start starts the metrics server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("starting metrics server", zap.String("addr", s.server.Addr))

	errCh := make(chan error, 1)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server failed: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.Stop(ctx)
	}
}

// Stop gracefully stops the metrics server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping metrics server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("failed to gracefully shutdown metrics server", zap.Error(err))
		return err
	}

	s.logger.Info("metrics server stopped")
	return nil
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.server.Addr
}
