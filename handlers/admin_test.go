package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"xmen-uploader/config"
	"xmen-uploader/db"
)

func TestAdminStatsHandler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uploader-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config.AppConfig.DatabasePath = filepath.Join(tmpDir, "test.db")
	config.AppConfig.StoragePath = filepath.Join(tmpDir, "uploads")
	config.AppConfig.AdminSecret = "super-secret-key-1234"
	db.InitDB()

	t.Run("rejects request without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/stats", nil)
		rec := httptest.NewRecorder()
		AdminStatsHandler(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("rejects request with wrong bearer token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/stats", nil)
		req.Header.Set("Authorization", "Bearer wrong-password")
		rec := httptest.NewRecorder()
		AdminStatsHandler(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("accepts request with valid bearer token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/stats", nil)
		req.Header.Set("Authorization", "Bearer super-secret-key-1234")
		rec := httptest.NewRecorder()
		AdminStatsHandler(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("accepts request with valid X-Admin-Secret header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/stats", nil)
		req.Header.Set("X-Admin-Secret", "super-secret-key-1234")
		rec := httptest.NewRecorder()
		AdminStatsHandler(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("rejects when admin secret is unconfigured", func(t *testing.T) {
		orig := config.AppConfig.AdminSecret
		config.AppConfig.AdminSecret = ""
		defer func() { config.AppConfig.AdminSecret = orig }()

		req := httptest.NewRequest("GET", "/api/admin/stats", nil)
		req.Header.Set("Authorization", "Bearer ")
		rec := httptest.NewRecorder()
		AdminStatsHandler(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})
}
