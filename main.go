package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"xmen-uploader/config"
	"xmen-uploader/db"
	"xmen-uploader/handlers"
)

//go:embed static
var staticFS embed.FS

var Version = "dev"

func main() {
	config.LoadConfig()
	db.InitDB()
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		startCleanupTask(ctx)
	}()

	r := chi.NewRouter()

	if config.AppConfig.TrustProxy {
		r.Use(middleware.RealIP)
	}

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none';")
			next.ServeHTTP(w, r)
		})
	})

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.AppConfig.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Admin-Secret"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Use(httprate.LimitByIP(60, 1*time.Minute))

	r.Route("/api", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				next.ServeHTTP(w, r)
			})
		})

		r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"max_upload_size_bytes": config.AppConfig.MaxUploadSize,
				"max_lifetime_days":     config.AppConfig.MaxAllowedLifetimeDays,
			})
		})

		r.With(httprate.LimitByIP(10, 1*time.Minute)).Post("/upload", handlers.UploadHandler)
		r.With(httprate.LimitByIP(20, 1*time.Minute)).Get("/download/{id}", handlers.DownloadHandler)
		r.With(httprate.LimitByIP(30, 1*time.Minute)).Get("/info/{id}", handlers.GetFileInfoHandler)
		r.With(httprate.LimitByIP(5, 1*time.Minute)).Get("/admin/stats", handlers.AdminStatsHandler)
	})

	serveDownloadPage := func(w http.ResponseWriter, r *http.Request) {
		content, _ := staticFS.ReadFile("static/download.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(content)
	}
	r.Get("/f/{id}", serveDownloadPage)
	r.Head("/f/{id}", serveDownloadPage)

	serveAdminPage := func(w http.ResponseWriter, r *http.Request) {
		content, _ := staticFS.ReadFile("static/admin.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(content)
	}
	r.Get("/admin", serveAdminPage)
	r.Head("/admin", serveAdminPage)

	staticDir, _ := fs.Sub(staticFS, "static")
	FileServer(r, "/", http.FS(staticDir))

	server := &http.Server{
		Addr:              ":" + config.AppConfig.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		log.Printf("Starting XMen Uploader %s on port %s", Version, config.AppConfig.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}

	<-cleanupDone
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	base := strings.TrimSuffix(path, "/") + "/"
	handler := http.StripPrefix(base, http.FileServer(root))
	pattern := base + "*"

	r.Get(pattern, handler.ServeHTTP)
	r.Head(pattern, handler.ServeHTTP)
}

func startCleanupTask(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	runCleanup()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCleanup()
		}
	}
}

func runCleanup() {
	files, err := db.GetExpiredFiles()
	if err != nil {
		log.Printf("Cleanup error: %v", err)
		return
	}

	for _, f := range files {
		storagePath := filepath.Join(config.AppConfig.StoragePath, filepath.Base(f.StorageName))
		if err := os.Remove(storagePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Printf("Cleanup: failed to remove file %s: %v", f.ID, err)
			continue
		}
		if err := db.DeleteFile(f.ID); err != nil {
			log.Printf("Cleanup: failed to delete database row %s: %v", f.ID, err)
			continue
		}
		log.Printf("Cleaned up expired file: %s", f.ID)
	}
}
