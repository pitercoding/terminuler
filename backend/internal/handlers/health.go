package handlers

import (
	"context"
	"log"
	"net/http"
	"time"
)

// healthPingTimeout keeps the health check fast when the database hangs:
// the hosting platform treats a slow answer as a failure anyway.
const healthPingTimeout = 2 * time.Second

// Pinger is the part of *sql.DB the health check needs.
type Pinger interface {
	PingContext(ctx context.Context) error
}

type healthResponse struct {
	Status string `json:"status"`
}

// NewHealthHandler answers 200 {"status":"ok"} when the database answers a
// ping and 503 {"status":"unavailable"} otherwise, so the hosting platform's
// health check notices an API that is up but cannot serve any booking. The
// database error is only logged, never returned.
func NewHealthHandler(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthPingTimeout)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			log.Printf("health check: database ping failed: %v", err)

			writeJSON(w, http.StatusServiceUnavailable, healthResponse{
				Status: "unavailable",
			})
			return
		}

		writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
	}
}
