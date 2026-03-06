package server

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ivmm/rpmmanager/internal/auth"
	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/database"
	embeddedFS "github.com/ivmm/rpmmanager/internal/embed"
	"github.com/ivmm/rpmmanager/internal/handler"
	"github.com/ivmm/rpmmanager/internal/service"
)

type Server struct {
	cfg          *config.Config
	db           *sql.DB
	router       http.Handler
	monitorSvc   *service.MonitorService
	rateLimiter  *auth.RateLimiter
	challengeSvc *auth.ChallengeService
}

func New(cfg *config.Config) (*Server, error) {
	// Setup logging
	setupLogging(cfg.Log)

	// Open database
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Auth service
	authService := auth.NewService(&cfg.Auth)

	// Frontend FS
	var frontendFS fs.FS
	distFS, err := fs.Sub(embeddedFS.FrontendFS, "dist")
	if err != nil {
		log.Warn().Msg("Frontend assets not embedded; running in API-only mode")
	} else {
		frontendFS = distFS
	}

	// Build router
	result := handler.NewRouter(&handler.Deps{
		Config:      cfg,
		DB:          db,
		AuthService: authService,
		FrontendFS:  frontendFS,
	})

	return &Server{
		cfg:          cfg,
		db:           db,
		router:       result.Handler,
		monitorSvc:   result.MonitorSvc,
		rateLimiter:  result.RateLimiter,
		challengeSvc: result.ChallengeSvc,
	}, nil
}

func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         s.cfg.Server.Listen,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Str("listen", s.cfg.Server.Listen).Msg("Starting RPM Manager server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	<-stop
	log.Info().Msg("Shutting down server...")

	// Stop background services
	if s.monitorSvc != nil {
		s.monitorSvc.Stop()
	}
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
	if s.challengeSvc != nil {
		s.challengeSvc.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	s.db.Close()
	log.Info().Msg("Server stopped")
	return nil
}

func setupLogging(cfg config.LogConfig) {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.Format == "text" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
}
