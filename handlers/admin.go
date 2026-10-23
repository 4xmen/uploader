package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"xmen-uploader/config"
	"xmen-uploader/db"
)

func AdminStatsHandler(w http.ResponseWriter, r *http.Request) {
	configuredSecret := config.AppConfig.AdminSecret
	if configuredSecret == "" {
		http.Error(w, "Admin access is disabled", http.StatusForbidden)
		return
	}

	secret := extractSecret(r)
	if secret == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(configuredSecret)) != 1 {
		time.Sleep(100 * time.Millisecond)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stats, err := db.GetStats()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func extractSecret(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	if xSecret := r.Header.Get("X-Admin-Secret"); xSecret != "" {
		return strings.TrimSpace(xSecret)
	}

	return r.URL.Query().Get("secret")
}
