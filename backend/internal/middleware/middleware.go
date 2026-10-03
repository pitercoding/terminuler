// Package middleware holds the HTTP middlewares that wrap every route.
package middleware

import (
	"log"
	"net/http"
	"time"
)

// SecurityHeaders sets the headers every API response needs. The API only
// answers JSON, never HTML, so the policy forbids loading or framing
// anything, and no response is cached: admin responses hold customer data,
// and availability changes with every booking.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()

		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Cache-Control", "no-store")

		next.ServeHTTP(w, r)
	})
}

// AccessLog logs the method, path, status and duration of each request.
// It logs neither the query string nor any header, so session tokens, the
// proxy secret and customer data never reach the logs. Successful health
// checks, which the hosting platform sends every few seconds, are skipped.
func AccessLog(logger *log.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		if r.URL.Path == "/health" && recorder.status == http.StatusOK {
			return
		}

		logger.Printf(
			"%s %s %d %s",
			r.Method,
			r.URL.Path,
			recorder.status,
			time.Since(start).Round(time.Millisecond),
		)
	})
}

// statusRecorder remembers the status code written by the handler. A
// handler that writes the body without calling WriteHeader answers 200.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(status int) {
	if !s.wroteHeader {
		s.status = status
		s.wroteHeader = true
	}

	s.ResponseWriter.WriteHeader(status)
}

// Unwrap lets http.ResponseController reach the original writer, which
// http.MaxBytesReader relies on to close the connection of oversized bodies.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
