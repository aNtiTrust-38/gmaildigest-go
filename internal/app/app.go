package app

import (
	"context"
	"database/sql"
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
	"gmaildigest-go/internal/worker"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Application holds all the major components of the service.
type Application struct {
	Config        *config.Config
	Logger        *log.Logger
	Scheduler     *scheduler.Scheduler
	DB            *sql.DB
	Auth          *auth.OAuthManager
	SessionStore  session.Store
	HttpServer    *http.Server
	MetricsServer *http.Server
	WorkerPool    *worker.WorkerPool
}

// New creates and initializes a new Application instance.
func New(cfg *config.Config) (*Application, error) {
	logger := log.New(os.Stdout, "gmaildigest: ", log.LstdFlags)

	// Setup: Database
	db, err := sql.Open("sqlite3", cfg.DB.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storageInstance := storage.NewSQLiteStorage(db)
	if err := storageInstance.Migrate(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Setup: WorkerPool
	pool := worker.NewWorkerPool(cfg.Worker.NumWorkers)

	// Setup: Scheduler
	sched, err := scheduler.NewScheduler(context.Background(), db, pool)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	// Setup: Auth Manager
	tokenStore := storage.NewTokenStore(storageInstance, []byte(cfg.Auth.TokenEncryptionKey))
	pkceStore := auth.NewInMemoryPKCEStore()
	stateStore := auth.NewInMemoryStateStore()
	oauthManager := auth.NewOAuthManager(tokenStore, pkceStore, stateStore)
	if err := oauthManager.LoadCredentials(cfg.Auth.CredentialsPath); err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Setup: TokenRefreshService
	tokenRefreshService := auth.NewTokenRefreshService(oauthManager)

	// Register job handlers
	sched.RegisterHandler("token_refresh", tokenRefreshService.HandleTokenRefreshJob)

	// Setup: Session Store
	sessionStore := session.NewInMemoryStore()

	// Setup: HTTP Server for metrics
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.MetricsPort),
		Handler: metricsMux,
	}

	// Setup: Main HTTP Server
	httpMux := http.NewServeMux()
	// TODO: Add main application handlers to httpMux
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: httpMux,
	}

	app := &Application{
		Config:        cfg,
		Logger:        logger,
		DB:            db,
		WorkerPool:    pool,
		Scheduler:     sched,
		Auth:          oauthManager,
		SessionStore:  sessionStore,
		HttpServer:    httpServer,
		MetricsServer: metricsServer,
	}

	// Register HTTP handlers
	httpMux.HandleFunc("/login", app.handleLogin)
	httpMux.HandleFunc("/auth/callback", app.handleAuthCallback)
	httpMux.HandleFunc("/logout", app.handleLogout)

	// Protected routes
	httpMux.Handle("/dashboard", app.requireAuth(http.HandlerFunc(app.handleDashboard)))
	httpMux.Handle("/", app.requireAuth(http.RedirectHandler("/dashboard", http.StatusTemporaryRedirect)))

	return app, nil
}

// Start begins the application's services.
func (a *Application) Start(ctx context.Context) error {
	a.Logger.Println("Starting application services...")

	// Start the worker pool
	a.WorkerPool.Start()
	a.Logger.Println("Worker pool started.")

	// Start the scheduler
	a.Scheduler.Start()
	a.Logger.Println("Scheduler started.")

	// Start the metrics server
	go func() {
		a.Logger.Printf("Starting metrics server on %s", a.MetricsServer.Addr)
		if err := a.MetricsServer.ListenAndServe(); err != http.ErrServerClosed {
			a.Logger.Fatalf("Metrics server ListenAndServe: %v", err)
		}
	}()

	// Start the main HTTP server
	go func() {
		a.Logger.Printf("Starting HTTP server on %s", a.HttpServer.Addr)
		if err := a.HttpServer.ListenAndServe(); err != http.ErrServerClosed {
			a.Logger.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the application's services.
func (a *Application) Stop(ctx context.Context) error {
	a.Logger.Println("Stopping application services...")

	// Shutdown servers
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := a.HttpServer.Shutdown(shutdownCtx); err != nil {
		a.Logger.Printf("HTTP server shutdown error: %v", err)
	}

	if err := a.MetricsServer.Shutdown(shutdownCtx); err != nil {
		a.Logger.Printf("Metrics server shutdown error: %v", err)
	}

	// Stop the scheduler
	a.Scheduler.Stop()
	a.Logger.Println("Scheduler stopped.")

	// Stop the worker pool
	a.WorkerPool.Stop()
	a.Logger.Println("Worker pool stopped.")

	// Close the database connection
	if err := a.DB.Close(); err != nil {
		a.Logger.Printf("Error closing database: %v", err)
	}

	a.Logger.Println("Application stopped gracefully.")
	return nil
} 