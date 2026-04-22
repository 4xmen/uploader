package models

import "time"

type File struct {
	ID            string    `json:"id"`
	OriginalName  string    `json:"original_name"`
	StorageName   string    `json:"-"`
	Size          int64     `json:"size"`
	MimeType      string    `json:"mime_type"`
	UploadedAt    time.Time `json:"uploaded_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	MaxDownloads  int       `json:"max_downloads"`
	DownloadCount int       `json:"download_count"`
}
