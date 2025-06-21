package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gmaildigest-go/internal/auth"
	"gmaildigest-go/internal/config"
	"gmaildigest-go/internal/scheduler"
	"gmaildigest-go/internal/session"
	"gmaildigest-go/internal/storage"
	"gmaildigest-go/internal/telegram"
	"gmaildigest-go/internal/worker"

	"github.com/go-co-op/gocron/v2"
)

// Application holds the application's dependencies
type Application struct {
	logger          *log.Logger
	config          *config.Config
	server          *http.Server
	authService     *auth.AuthService
	sessionStore    session.Store
	storage         storage.Storage
	scheduler       gocron.Scheduler
	workerPool      *worker.Pool
	telegramService *telegram.Service
}

// New creates a new Application.
func New(cfg *config.Config) (*Application, error) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := storage.NewSQLiteStorage(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	if err := db.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	authService, err := auth.New(
		cfg.Auth.ClientID,
		cfg.Auth.ClientSecret,
		fmt.Sprintf("http://localhost:%d/auth/callback", cfg.HTTPPort),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth service: %w", err)
	}

	sessionStore := session.NewInMemoryStore()
	workerPool := worker.NewPool(cfg.NumWorkers)

	telegramService, err := telegram.NewService(cfg.Telegram.BotToken, cfg.HTTPPort, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram service: %w", err)
	}

	app := &Application{
		logger:          logger,
		config:          cfg,
		authService:     authService,
		sessionStore:    sessionStore,
		storage:         db,
		workerPool:      workerPool,
		telegramService: telegramService,
	}

	app.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: app.routes(),
	}

	s, err := scheduler.New(logger, app.workerPool, app.storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}
	app.scheduler = s

	return app, nil
}

// Run starts the application.
func (a *Application) Run() error {
	a.logger.Printf("Starting server on %s", a.server.Addr)
	go a.telegramService.StartPolling()
	a.workerPool.Start()
	a.scheduler.Start()
	return a.server.ListenAndServe()
}

// Shutdown gracefully shuts down the application.
func (a *Application) Shutdown(ctx context.Context) error {
	a.logger.Println("Shutting down server...")
	a.workerPool.Stop()
	if err := a.scheduler.Shutdown(); err != nil {
		a.logger.Printf("Error shutting down scheduler: %v", err)
	}
	return a.server.Shutdown(ctx)
}

func (a *Application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /login", a.handleLogin)
	mux.HandleFunc("GET /auth/callback", a.handleAuthCallback)
	mux.HandleFunc("POST /logout", a.handleLogout)

	// Authenticated routes
	mux.Handle("GET /", a.requireAuth(http.HandlerFunc(a.handleDashboard)))
	mux.Handle("GET /dashboard", a.requireAuth(http.HandlerFunc(a.handleDashboard)))
	mux.Handle("GET /telegram/connect", a.requireAuth(http.HandlerFunc(a.handleTelegramConnect)))

	return mux
} 