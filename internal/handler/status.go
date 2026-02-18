package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatusHandler struct {
	db *pgxpool.Pool
}

func NewStatusHandler(db *pgxpool.Pool) *StatusHandler {
	return &StatusHandler{db: db}
}

// Live handles GET /health/live
// Prüft nur ob der Prozess läuft — kein DB-Ping.
// Wird von K8s livenessProbe verwendet.
func (s *StatusHandler) Live(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
}

// Ready handles GET /health/ready
// Prüft ob die Datenbankverbindung aktiv ist.
// Wird von K8s readinessProbe verwendet.
func (s *StatusHandler) Ready(w http.ResponseWriter, r *http.Request) {
	pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(pingCtx); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unavailable","reason":"database unreachable"}`)) //nolint:errcheck
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`)) //nolint:errcheck
}
