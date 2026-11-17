package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
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
		if errors.Is(err, sql.ErrNoRows) {
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

	safeFileName := filepath.Base(file.StorageName)
	storagePath := filepath.Join(config.AppConfig.StoragePath, safeFileName)

	// Open the file before consuming a download so missing files do not count,
	// and so cleanup cannot unlink the file between the check and the response.
	f, err := os.Open(storagePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "File missing on disk", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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

	cd := mime.FormatMediaType("attachment", map[string]string{"filename": file.OriginalName})
	if cd == "" {
		cd = "attachment"
	}
	w.Header().Set("Content-Disposition", cd)
	// Always serve downloads as opaque bytes. Content-Disposition already forces a
	// download, and this keeps an uploaded HTML/SVG file from ever rendering inline.
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, file.OriginalName, info.ModTime(), f)
}

func GetFileInfoHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	file, err := db.GetFile(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(file)
}
