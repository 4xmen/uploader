package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	r.Body = http.MaxBytesReader(w, r.Body, config.AppConfig.MaxUploadSize)

	err := r.ParseMultipartForm(config.AppConfig.MaxUploadSize)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, false, "File exceeds maximum upload size or invalid request", nil)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, false, "Error retrieving the file", nil)
		return
	}
	defer file.Close()

	maxDownloads, _ := strconv.Atoi(r.FormValue("max_downloads"))
	if maxDownloads < 0 {
		maxDownloads = 0
	} else if maxDownloads > 10000 {
		maxDownloads = 10000
	}

	lifetimeHours, _ := strconv.Atoi(r.FormValue("lifetime_hours"))
	if lifetimeHours <= 0 {
		lifetimeHours = 24
	}
	maxAllowedHours := config.AppConfig.MaxAllowedLifetimeDays * 24
	if lifetimeHours > maxAllowedHours {
		sendJSONResponse(w, http.StatusBadRequest, false, "Lifetime exceeds the maximum allowed limit", nil)
		return
	}

	expiresAt := time.Now().Add(time.Duration(lifetimeHours) * time.Hour)

	id, err := gonanoid.New(10)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, false, "Failed to generate file identifier", nil)
		return
	}

	ext := sanitizeExtension(handler.Filename)
	storageName := id + ext
	storagePath := filepath.Join(config.AppConfig.StoragePath, storageName)

	dst, err := os.Create(storagePath)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error saving file to disk", nil)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(storagePath)
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error copying file content", nil)
		return
	}

	cleanOriginalName := filepath.Base(handler.Filename)
	if cleanOriginalName == "." || cleanOriginalName == "/" || cleanOriginalName == "" {
		cleanOriginalName = "file"
	}

	fileModel := &models.File{
		ID:            id,
		OriginalName:  cleanOriginalName,
		StorageName:   storageName,
		Size:          handler.Size,
		MimeType:      handler.Header.Get("Content-Type"),
		UploadedAt:    time.Now(),
		ExpiresAt:     expiresAt,
		MaxDownloads:  maxDownloads,
		DownloadCount: 0,
	}

	if err := db.SaveFile(fileModel); err != nil {
		os.Remove(storagePath)
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error saving to database", nil)
		return
	}

	sendJSONResponse(w, http.StatusOK, true, "File uploaded successfully", fileModel)
}

func sanitizeExtension(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) > 16 || strings.ContainsAny(ext, "/\\ \x00;\"'") {
		return ""
	}
	return ext
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
