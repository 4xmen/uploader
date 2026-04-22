package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"xmen-uploader/config"
	"xmen-uploader/db"
	"xmen-uploader/handlers"
)

func main() {
	// Initialize Configuration
	config.LoadConfig()

	// Initialize Database
	db.InitDB()

	// Start Background Cleanup Task
	go startCleanupTask()

	// Initialize Router
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
	}))

	// Global rate limit: 100 requests per minute per IP
	r.Use(httprate.LimitByIP(100, 1*time.Minute))

	// API Routes
	r.Route("/api", func(r chi.Router) {
		// Public config endpoint
		r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"max_upload_size_bytes": config.AppConfig.MaxUploadSize,
				"max_lifetime_days":     config.AppConfig.MaxAllowedLifetimeDays,
			})
		})

		// Stricter rate limit for uploads: 10 uploads per minute per IP
		r.With(httprate.LimitByIP(10, 1*time.Minute)).Post("/upload", handlers.UploadHandler)
		
		r.Get("/download/{id}", handlers.DownloadHandler)
		r.Get("/info/{id}", handlers.GetFileInfoHandler)
		r.Get("/admin/stats", handlers.AdminStatsHandler)
	})

	// Frontend routes
	workDir, _ := os.Getwd()
	r.Get("/f/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(workDir, "static", "download.html"))
	})
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(workDir, "static", "admin.html"))
	})

	// Serve Static Files (CSS, JS, index.html)
	filesDir := http.Dir(filepath.Join(workDir, "static"))
	FileServer(r, "/", filesDir)

	// Start Server
	log.Printf("Starting XMen Uploader on port %s", config.AppConfig.Port)
	err := http.ListenAndServe(":"+config.AppConfig.Port, r)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}

// Background task to delete expired or fully downloaded files
func startCleanupTask() {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		<-ticker.C
		files, err := db.GetExpiredFiles()
		if err != nil {
			log.Printf("Cleanup error: %v", err)
			continue
		}

		for _, f := range files {
			// Delete physical file
			storagePath := filepath.Join(config.AppConfig.StoragePath, f.StorageName)
			os.Remove(storagePath)

			// Delete from DB
			db.DeleteFile(f.ID)
			log.Printf("Cleaned up expired file: %s", f.ID)
		}
	}
}
