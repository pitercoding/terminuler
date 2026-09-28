package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pitercoding/terminuler/internal/config"
	"github.com/pitercoding/terminuler/internal/database"
	"github.com/pitercoding/terminuler/internal/email"
	"github.com/pitercoding/terminuler/internal/handlers"
	"github.com/pitercoding/terminuler/internal/repositories"
	"github.com/pitercoding/terminuler/internal/routes"
	"github.com/pitercoding/terminuler/internal/services"
)

const (
	// readHeaderTimeout limits how long a client may take to send the request headers, so slow clients cannot hold connections open.
	readHeaderTimeout = 5 * time.Second

	// readTimeout limits reading the whole request, including the body.
	readTimeout = 10 * time.Second

	// writeTimeout must exceed the 10 second timeout of the confirmation email, which is sent before the booking response is written.
	writeTimeout = 20 * time.Second

	// idleTimeout limits how long keep-alive connections stay open between requests.
	idleTimeout = 60 * time.Second

	// shutdownTimeout is how long in-flight requests have to finish after a shutdown signal before the server stops anyway.
	shutdownTimeout = 15 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run starts the API and blocks until it stops. Returning instead of calling log.Fatal lets deferred cleanup, such as closing the database, run.
func run() error {
	if err := config.Load(); err != nil {
		return err
	}

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		return err
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	location, err := config.Location()
	if err != nil {
		return err
	}

	resendAPIKey, err := config.ResendAPIKey()
	if err != nil {
		return err
	}

	resendFromEmail := config.ResendFromEmail()

	mux := http.NewServeMux()

	appointmentRepository := repositories.NewAppointmentRepository(db)

	emailSender := email.NewResendSender(
		resendAPIKey,
		resendFromEmail,
	)

	appointmentService := services.NewAppointmentServiceWithEmail(
		appointmentRepository,
		emailSender,
		func() time.Time {
			return time.Now().In(location)
		},
	)

	appointmentHandler := handlers.NewAppointmentHandler(
		appointmentService,
	)

	routes.RegisterRoutes(
		mux,
		appointmentHandler,
	)

	port := config.Port()

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf(
			"Terminuler API running on http://localhost:%s (timezone: %s)",
			port,
			location,
		)

		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server failed: %w", err)
	case <-ctx.Done():
	}

	// Restore the default signal behavior: a second signal stops the process immediately instead of waiting for the graceful shutdown.
	stop()

	log.Println("Shutting down, waiting for in-flight requests to finish")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}

	log.Println("Server stopped")

	return nil
}
