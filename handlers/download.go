package handlers

import (
	"database/sql"
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"xmen-uploader/config"
	"xmen-uploader/db"
)

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	file, err := db.GetFile(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "File not found or expired", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if time.Now().After(file.ExpiresAt) {
		http.Error(w, "File expired", http.StatusGone)
		return
	}

	allowed, err := db.TryIncrementDownloadCount(id)
	if err != nil {
		http.Error(w, "Failed to update download stats", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Max downloads reached", http.StatusGone)
		return
	}

	safeFileName := filepath.Base(file.StorageName)
	storagePath := filepath.Join(config.AppConfig.StoragePath, safeFileName)
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		http.Error(w, "File missing on disk", http.StatusNotFound)
		return
	}

	cd := mime.FormatMediaType("attachment", map[string]string{"filename": file.OriginalName})
	w.Header().Set("Content-Disposition", cd)
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, storagePath)
}

func GetFileInfoHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	file, err := db.GetFile(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if time.Now().After(file.ExpiresAt) {
		http.Error(w, "File expired", http.StatusGone)
		return
	}

	if file.MaxDownloads > 0 && file.DownloadCount >= file.MaxDownloads {
		http.Error(w, "Max downloads reached", http.StatusGone)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}
