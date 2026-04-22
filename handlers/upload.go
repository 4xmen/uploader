package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"xmen-uploader/config"
	"xmen-uploader/db"
	"xmen-uploader/models"
)

type UploadResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	File    *models.File `json:"file,omitempty"`
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, config.AppConfig.MaxUploadSize)

	err := r.ParseMultipartForm(config.AppConfig.MaxUploadSize)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, false, "File too large or invalid request", nil)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, false, "Error retrieving the file", nil)
		return
	}
	defer file.Close()

	// Parse custom options
	maxDownloadsStr := r.FormValue("max_downloads")
	lifetimeHoursStr := r.FormValue("lifetime_hours")

	maxDownloads, _ := strconv.Atoi(maxDownloadsStr)
	lifetimeHours, _ := strconv.Atoi(lifetimeHoursStr)

	if lifetimeHours <= 0 {
		lifetimeHours = 24 // default 1 day
	}
	maxAllowedHours := config.AppConfig.MaxAllowedLifetimeDays * 24
	if lifetimeHours > maxAllowedHours {
		lifetimeHours = maxAllowedHours
	}

	expiresAt := time.Now().Add(time.Duration(lifetimeHours) * time.Hour)

	// Generate ID and Storage Name
	id, _ := gonanoid.New(10)
	storageName := id + filepath.Ext(handler.Filename)
	storagePath := filepath.Join(config.AppConfig.StoragePath, storageName)

	// Save to disk
	dst, err := os.Create(storagePath)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error saving file", nil)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error copying file", nil)
		return
	}

	// Create model
	fileModel := &models.File{
		ID:            id,
		OriginalName:  handler.Filename,
		StorageName:   storageName,
		Size:          handler.Size,
		MimeType:      handler.Header.Get("Content-Type"),
		UploadedAt:    time.Now(),
		ExpiresAt:     expiresAt,
		MaxDownloads:  maxDownloads,
		DownloadCount: 0,
	}

	// Save to DB
	if err := db.SaveFile(fileModel); err != nil {
		// Cleanup physical file if db fails
		os.Remove(storagePath)
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error saving to database", nil)
		return
	}

	sendJSONResponse(w, http.StatusOK, true, "File uploaded successfully", fileModel)
}

func sendJSONResponse(w http.ResponseWriter, status int, success bool, message string, file *models.File) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(UploadResponse{
		Success: success,
		Message: message,
		File:    file,
	})
}
