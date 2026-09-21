package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"remote-wol/internal/agent"
	"remote-wol/internal/auth"
	"remote-wol/internal/config"
	"remote-wol/internal/db"
	"remote-wol/internal/handlers"
	"remote-wol/internal/middleware"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	cfg := config.Load()

	// Initialize database
	database, err := db.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize auth
	authService := auth.New(cfg.JWTSecret, cfg.AdminUser, cfg.AdminPass)

	// Initialize agent client
	agentClient := agent.NewClient(cfg.AgentURL, cfg.AgentToken)

	// Initialize handlers
	deviceHandler := handlers.NewDeviceHandler(database, agentClient)

	// Setup router
	mux := http.NewServeMux()

	// Auth route (no JWT required)
	mux.HandleFunc("/api/auth/login", authService.LoginHandler)

	// Protected API routes
	protected := authService.Middleware(deviceHandler)
	mux.Handle("/api/devices/", protected)
	mux.Handle("/api/devices", protected)

	// Serve embedded frontend files
	publicContent, err := fs.Sub(publicFS, "public")
	if err != nil {
		log.Fatalf("Failed to access embedded public files: %v", err)
	}
	fileServer := http.FileServer(http.FS(publicContent))

	// SPA fallback: serve index.html for non-API, non-file routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file first
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if the file exists in the embedded FS
		if f, err := publicContent.Open(path[1:]); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	// Apply global middleware
	handler := middleware.Chain(mux,
		middleware.Recovery,
		middleware.Logger,
		middleware.CORS,
	)

	// Create server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("🚀 Remote WoL server starting on http://0.0.0.0:%s", cfg.Port)
	log.Printf("📡 Agent endpoint: %s", cfg.AgentURL)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped.")
}
