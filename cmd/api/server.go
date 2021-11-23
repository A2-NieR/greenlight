package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Shutdown error channel receives any errors returned by Shutdown()
	shutdownError := make(chan error)

	go func() {
		// Quit channel carries os.Signal values
		quit := make(chan os.Signal, 1)

		// Listen for incoming SIGINT & SIGTERM signals and relay to quit channel
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Read signal from quit channel, blocks until signal is received
		s := <-quit

		// Log caught signal message
		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		// Create context with 5s timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Call Shutdown(), returns nil if successful or error if not, relay to shutdownError channel
		shutdownError <- srv.Shutdown(ctx)
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	// Check for http.ErrServerClosed error => shutdown has started
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Wait for return value from Shutdown() on error channel
	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
