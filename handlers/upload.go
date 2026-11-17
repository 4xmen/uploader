package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
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

const (
	// Files are streamed to disk instead of being buffered in memory.
	multipartMemory = 8 << 20
	// Multipart boundaries and part headers are not part of the file size limit.
	multipartOverhead = 1 << 20
)

type UploadResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	File    *models.File `json:"file,omitempty"`
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	maxUploadSize := config.AppConfig.MaxUploadSize
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+multipartOverhead)

	parseErr := r.ParseMultipartForm(multipartMemory)
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}
	if parseErr != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(parseErr, &maxBytesErr) {
			sendJSONResponse(w, http.StatusRequestEntityTooLarge, false, "File exceeds maximum upload size", nil)
			return
		}
		sendJSONResponse(w, http.StatusBadRequest, false, "Invalid upload request", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, false, "No file was provided", nil)
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxUploadSize {
		sendJSONResponse(w, http.StatusRequestEntityTooLarge, false, "File exceeds maximum upload size", nil)
		return
	}

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

	now := time.Now()
	expiresAt := now.Add(time.Duration(lifetimeHours) * time.Hour)

	id, err := gonanoid.New(10)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, false, "Failed to generate file identifier", nil)
		return
	}

	ext := sanitizeExtension(header.Filename)
	storageName := id + ext
	storagePath := filepath.Join(config.AppConfig.StoragePath, storageName)

	dst, err := os.OpenFile(storagePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error saving file to disk", nil)
		return
	}

	n, copyErr := io.Copy(dst, io.LimitReader(file, maxUploadSize+1))
	if copyErr != nil {
		_ = dst.Close()
		_ = os.Remove(storagePath)
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error copying file content", nil)
		return
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(storagePath)
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error saving file to disk", nil)
		return
	}
	if n > maxUploadSize {
		_ = os.Remove(storagePath)
		sendJSONResponse(w, http.StatusRequestEntityTooLarge, false, "File exceeds maximum upload size", nil)
		return
	}

	fileModel := &models.File{
		ID:            id,
		OriginalName:  cleanOriginalName(header.Filename),
		StorageName:   storageName,
		Size:          n,
		MimeType:      normalizeMimeType(header.Header.Get("Content-Type")),
		UploadedAt:    now,
		ExpiresAt:     expiresAt,
		MaxDownloads:  maxDownloads,
		DownloadCount: 0,
	}

	if err := db.SaveFile(fileModel); err != nil {
		_ = os.Remove(storagePath)
		sendJSONResponse(w, http.StatusInternalServerError, false, "Error saving to database", nil)
		return
	}

	sendJSONResponse(w, http.StatusOK, true, "File uploaded successfully", fileModel)
}

func cleanOriginalName(filename string) string {
	name := filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	name = strings.TrimSpace(name)
	switch name {
	case "", ".", "..", "/":
		return "file"
	}
	return name
}

func sanitizeExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" || ext == "." || len(ext) > 16 || strings.ContainsAny(ext, "/\\ \x00\r\n\t;\"'") {
		return ""
	}
	return ext
}

func normalizeMimeType(value string) string {
	if value == "" {
		return "application/octet-stream"
	}
	if _, _, err := mime.ParseMediaType(value); err != nil {
		return "application/octet-stream"
	}
	return value
}

func sendJSONResponse(w http.ResponseWriter, status int, success bool, message string, file *models.File) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(UploadResponse{
		Success: success,
		Message: message,
		File:    file,
	})
}
