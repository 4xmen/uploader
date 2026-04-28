package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"xmen-uploader/config"
	"xmen-uploader/db"
)

func AdminStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Simple secret token auth
	secret := r.URL.Query().Get("secret")
	if subtle.ConstantTimeCompare([]byte(secret), []byte(config.AppConfig.AdminSecret)) != 1 {
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
