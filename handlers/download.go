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

	// Check expiration
	if time.Now().After(file.ExpiresAt) {
		http.Error(w, "File expired", http.StatusGone)
		return
	}

	// Check max downloads
	if file.MaxDownloads > 0 && file.DownloadCount >= file.MaxDownloads {
		http.Error(w, "Max downloads reached", http.StatusGone)
		return
	}

	// Increment download count
	if err := db.IncrementDownloadCount(id); err != nil {
		http.Error(w, "Failed to update stats", http.StatusInternalServerError)
		return
	}

	// Serve the file (prevent directory traversal by strictly using filepath.Base)
	safeFileName := filepath.Base(file.StorageName)
	storagePath := filepath.Join(config.AppConfig.StoragePath, safeFileName)
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		http.Error(w, "File missing on disk", http.StatusNotFound)
		return
	}

	cd := mime.FormatMediaType("attachment", map[string]string{"filename": file.OriginalName})
	w.Header().Set("Content-Disposition", cd)
	w.Header().Set("Content-Type", file.MimeType)
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

	// Check expiration
	if time.Now().After(file.ExpiresAt) {
		http.Error(w, "File expired", http.StatusGone)
		return
	}

	// Check max downloads
	if file.MaxDownloads > 0 && file.DownloadCount >= file.MaxDownloads {
		http.Error(w, "Max downloads reached", http.StatusGone)
		return
	}

	// Hide exact storage path for security, and just send the safe metadata
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}
