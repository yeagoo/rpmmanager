package handler

import (
	"database/sql"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/ivmm/rpmmanager/internal/auth"
	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/repository"
	"github.com/ivmm/rpmmanager/internal/service"
)

type Deps struct {
	Config      *config.Config
	DB          *sql.DB
	AuthService *auth.Service
	FrontendFS  fs.FS // embedded frontend dist/
}

// RouterResult holds the router and cleanup functions.
type RouterResult struct {
	Handler    http.Handler
	MonitorSvc *service.MonitorService
}

func NewRouter(deps *Deps) *RouterResult {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Compress(5))

	// CORS: restrict origins based on config base_url, not wildcard
	allowedOrigins := []string{deps.Config.Server.BaseURL}
	if deps.Config.Server.BaseURL == "" || deps.Config.Server.BaseURL == "http://localhost:8080" {
		// Development mode: allow common dev origins
		allowedOrigins = []string{"http://localhost:5173", "http://localhost:8080"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Request body size limit (10MB)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
			next.ServeHTTP(w, r)
		})
	})

	// Services
	productRepo := repository.NewProductRepo(deps.DB)
	buildRepo := repository.NewBuildRepo(deps.DB)
	gpgKeyRepo := repository.NewGPGKeyRepo(deps.DB)
	settingsRepo := repository.NewSettingsRepo(deps.DB)
	monitorRepo := repository.NewMonitorRepo(deps.DB)
	productSvc := service.NewProductService(productRepo, monitorRepo)
	notificationSvc := service.NewNotificationService(settingsRepo)
	buildSvc := service.NewBuildService(deps.Config, buildRepo, productRepo, gpgKeyRepo, settingsRepo, notificationSvc)
	gpgSvc := service.NewGPGService(deps.Config, gpgKeyRepo)
	repoSvc := service.NewRepoService(deps.Config)
	monitorSvc := service.NewMonitorService(deps.Config, monitorRepo, settingsRepo, buildSvc)

	// Start background monitor
	monitorSvc.Start()

	// Repo RPM service
	repoRPMSvc := service.NewRepoRPMService(deps.Config, gpgSvc)

	// Handlers
	authHandler := NewAuthHandler(deps.AuthService)
	productHandler := NewProductHandler(productSvc)
	buildHandler := NewBuildHandler(deps.Config, buildSvc)
	wsHandler := NewWSHandler(deps.Config, buildSvc, deps.AuthService)
	gpgHandler := NewGPGHandler(gpgSvc)
	repoHandler := NewRepoHandler(repoSvc)
	repoRPMHandler := NewRepoRPMHandler(deps.Config, repoRPMSvc, productSvc)
	dashboardHandler := NewDashboardHandler(deps.DB)
	settingsHandler := NewSettingsHandler(settingsRepo)
	monitorHandler := NewMonitorHandler(monitorSvc)
	webhookHandler := NewWebhookHandler(buildSvc, productSvc)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		})
		r.Post("/api/auth/login", authHandler.Login)

		// Public repo RPM download (matches Caddy file path: /{product}/repo-rpm/{filename})
		r.Get("/{product}/repo-rpm/{filename}", repoRPMHandler.PublicDownload)
	})

	// Protected API routes
	r.Group(func(r chi.Router) {
		r.Use(deps.AuthService.Middleware)

		// Auth
		r.Get("/api/auth/me", authHandler.Me)

		// Products
		r.Get("/api/products", productHandler.List)
		r.Post("/api/products", productHandler.Create)
		r.Get("/api/products/{id}", productHandler.Get)
		r.Put("/api/products/{id}", productHandler.Update)
		r.Delete("/api/products/{id}", productHandler.Delete)
		r.Post("/api/products/{id}/duplicate", productHandler.Duplicate)
		r.Get("/api/products/{id}/export", productHandler.Export)
		r.Get("/api/products/export", productHandler.ExportAll)
		r.Post("/api/products/import", productHandler.Import)
		r.Post("/api/products/{id}/repo-rpm", repoRPMHandler.Generate)
		r.Get("/api/products/{id}/repo-rpm", repoRPMHandler.GetLatest)
		r.Get("/api/products/{id}/repo-rpm/download", repoRPMHandler.Download)

		// Distro metadata (for UI)
		r.Get("/api/distros", productHandler.GetDistros)

		// Builds
		r.Get("/api/builds", buildHandler.List)
		r.Post("/api/builds", buildHandler.Trigger)
		r.Get("/api/builds/{id}", buildHandler.Get)
		r.Post("/api/builds/{id}/cancel", buildHandler.Cancel)
		r.Get("/api/builds/{id}/log", buildHandler.GetLog)

		// WebSocket (build log streaming)
		r.Get("/api/builds/{id}/ws", wsHandler.BuildLog)

		// GPG Keys
		r.Get("/api/gpg-keys", gpgHandler.List)
		r.Post("/api/gpg-keys/import", gpgHandler.Import)
		r.Post("/api/gpg-keys/generate", gpgHandler.Generate)
		r.Get("/api/gpg-keys/{id}", gpgHandler.Get)
		r.Delete("/api/gpg-keys/{id}", gpgHandler.Delete)
		r.Get("/api/gpg-keys/{id}/export", gpgHandler.Export)
		r.Post("/api/gpg-keys/{id}/default", gpgHandler.SetDefault)

		// Repositories
		r.Get("/api/repos", repoHandler.ListProducts)
		r.Get("/api/repos/{product}/tree", repoHandler.GetTree)
		r.Get("/api/repos/{product}/file", repoHandler.GetFileContent)
		r.Get("/api/repos/{product}/rollbacks", repoHandler.ListRollbacks)
		r.Post("/api/repos/{product}/rollback", repoHandler.Rollback)

		// Dashboard
		r.Get("/api/dashboard", dashboardHandler.Get)

		// Settings
		r.Get("/api/settings", settingsHandler.GetAll)
		r.Put("/api/settings", settingsHandler.Update)

		// Monitors
		r.Get("/api/monitors", monitorHandler.List)
		r.Get("/api/monitors/{product_id}", monitorHandler.Get)
		r.Put("/api/monitors/{product_id}", monitorHandler.Update)
		r.Post("/api/monitors/{product_id}/check", monitorHandler.CheckNow)
	})

	// Webhook (API token auth, not JWT)
	r.Group(func(r chi.Router) {
		r.Use(deps.AuthService.Middleware)
		r.Post("/api/webhook/{product}", webhookHandler.Trigger)
	})

	// Serve frontend SPA
	if deps.FrontendFS != nil {
		serveSPA(r, deps.FrontendFS)
	}

	return &RouterResult{
		Handler:    r,
		MonitorSvc: monitorSvc,
	}
}

func serveSPA(r chi.Router, frontendFS fs.FS) {
	fileServer := http.FileServer(http.FS(frontendFS))

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to serve the file directly
		if _, err := fs.Stat(frontendFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
