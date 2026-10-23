package main

import (
	"embed"
	"encoding/json"
	"io/fs"
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

//go:embed static
var staticFS embed.FS

func main() {
	config.LoadConfig()
	db.InitDB()
	go startCleanupTask()

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
		r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"max_upload_size_bytes": config.AppConfig.MaxUploadSize,
				"max_lifetime_days":     config.AppConfig.MaxAllowedLifetimeDays,
			})
		})

		r.With(httprate.LimitByIP(10, 1*time.Minute)).Post("/upload", handlers.UploadHandler)
		r.With(httprate.LimitByIP(20, 1*time.Minute)).Get("/download/{id}", handlers.DownloadHandler)
		r.With(httprate.LimitByIP(30, 1*time.Minute)).Get("/info/{id}", handlers.GetFileInfoHandler)

		r.With(httprate.LimitByIP(5, 1*time.Minute)).Route("/admin/stats", func(r chi.Router) {
			r.Get("/", handlers.AdminStatsHandler)
			r.Post("/", handlers.AdminStatsHandler)
		})
	})

	r.Get("/f/{id}", func(w http.ResponseWriter, r *http.Request) {
		content, _ := staticFS.ReadFile("static/download.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	})

	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		content, _ := staticFS.ReadFile("static/admin.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	})

	staticDir, _ := fs.Sub(staticFS, "static")
	FileServer(r, "/", http.FS(staticDir))

	log.Printf("Starting XMen Uploader on port %s", config.AppConfig.Port)
	err := http.ListenAndServe(":"+config.AppConfig.Port, r)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

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

func startCleanupTask() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		files, err := db.GetExpiredFiles()
		if err != nil {
			log.Printf("Cleanup error: %v", err)
			continue
		}

		for _, f := range files {
			storagePath := filepath.Join(config.AppConfig.StoragePath, f.StorageName)
			os.Remove(storagePath)
			db.DeleteFile(f.ID)
			log.Printf("Cleaned up expired file: %s", f.ID)
		}
	}
}
