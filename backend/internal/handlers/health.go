package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"contact-manager/internal/utils"
)

const healthCheckTimeout = 2 * time.Second

type HealthHandler struct {
	DB *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{DB: db}
}

// Health reports whether the process can reach its database. It uses the shared
// response envelope so monitoring sees the same shape as every other endpoint.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
	defer cancel()

	if err := h.DB.PingContext(ctx); err != nil {
		utils.WriteJSON(w, http.StatusServiceUnavailable,
			map[string]string{"status": "degraded", "db": "down"}, "Database unreachable")
		return
	}

	utils.WriteJSON(w, http.StatusOK,
		map[string]string{"status": "ok", "db": "connected"}, "Service healthy")
}
