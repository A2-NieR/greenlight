package main

import (
	"fmt"
	"net/http"

	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Deferred function always runs in event of a panic
		defer func() {
			// Use recover to check if panic occurred or not
			if err := recover(); err != nil {
				// Connection: close triggers HTTP server to close current connection after sent response
				rw.Header().Set("Connection", "close")
				// Normalize returned interface{} value of recover() & call serverErrorResponse() helper => Log error using custom Logger at ERROR level & send client a 500 error response
				app.serverErrorResponse(rw, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(rw, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	// Allow an average of 2 requests/s with a max of 4 requests in a single burst
	limiter := rate.NewLimiter(2, 4)

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Check if request is permitted
		if !limiter.Allow() {
			// Call helper to return 429 response
			app.rateLimitExceededResponse(rw, r)
			return
		}

		next.ServeHTTP(rw, r)
	})
}
