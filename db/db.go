package db

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
	"xmen-uploader/config"
	"xmen-uploader/models"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", config.AppConfig.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		original_name TEXT NOT NULL,
		storage_name TEXT NOT NULL,
		size INTEGER NOT NULL,
		mime_type TEXT NOT NULL,
		uploaded_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL,
		max_downloads INTEGER NOT NULL,
		download_count INTEGER NOT NULL DEFAULT 0
	);
	`
	_, err = DB.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
}

func SaveFile(f *models.File) error {
	query := `
		INSERT INTO files (id, original_name, storage_name, size, mime_type, uploaded_at, expires_at, max_downloads, download_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	uploadedAtStr := f.UploadedAt.Format(time.RFC3339)
	expiresAtStr := f.ExpiresAt.Format(time.RFC3339)

	_, err := DB.Exec(query, f.ID, f.OriginalName, f.StorageName, f.Size, f.MimeType, uploadedAtStr, expiresAtStr, f.MaxDownloads, f.DownloadCount)
	return err
}

func GetFile(id string) (*models.File, error) {
	query := `SELECT id, original_name, storage_name, size, mime_type, uploaded_at, expires_at, max_downloads, download_count FROM files WHERE id = ?`
	row := DB.QueryRow(query, id)

	f := &models.File{}
	var uploadedAtStr, expiresAtStr string
	err := row.Scan(&f.ID, &f.OriginalName, &f.StorageName, &f.Size, &f.MimeType, &uploadedAtStr, &expiresAtStr, &f.MaxDownloads, &f.DownloadCount)
	if err != nil {
		return nil, err
	}

	f.UploadedAt, _ = time.Parse(time.RFC3339, uploadedAtStr)
	f.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
	return f, nil
}

func TryIncrementDownloadCount(id string) (bool, error) {
	query := `UPDATE files SET download_count = download_count + 1 WHERE id = ? AND (max_downloads = 0 OR download_count < max_downloads)`
	res, err := DB.Exec(query, id)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func IncrementDownloadCount(id string) error {
	_, err := TryIncrementDownloadCount(id)
	return err
}

func DeleteFile(id string) error {
	query := `DELETE FROM files WHERE id = ?`
	_, err := DB.Exec(query, id)
	return err
}

func GetExpiredFiles() ([]models.File, error) {
	query := `SELECT id, storage_name FROM files WHERE expires_at <= ? OR (max_downloads > 0 AND download_count >= max_downloads)`
	rows, err := DB.Query(query, time.Now().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.File
	for rows.Next() {
		var f models.File
		if err := rows.Scan(&f.ID, &f.StorageName); err != nil {
			continue
		}
		files = append(files, f)
	}
	return files, nil
}

func GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalFiles int
	err := DB.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&totalFiles)
	if err != nil {
		return nil, err
	}

	var totalSize int64
	err = DB.QueryRow(`SELECT COALESCE(SUM(size), 0) FROM files`).Scan(&totalSize)
	if err != nil {
		return nil, err
	}

	var totalDownloads int
	err = DB.QueryRow(`SELECT COALESCE(SUM(download_count), 0) FROM files`).Scan(&totalDownloads)
	if err != nil {
		return nil, err
	}

	stats["active_files"] = totalFiles
	stats["total_size_bytes"] = totalSize
	stats["total_downloads"] = totalDownloads

	return stats, nil
}
