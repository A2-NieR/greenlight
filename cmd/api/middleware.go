package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

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
	// Client struct holds the rate limiter & last seen time for each client
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// Map holds client's IP addresses & rate limiters
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Background goroutine removing old entries from clients map once every minute
	go func() {
		for {
			time.Sleep(time.Minute)

			// Lock mutex preventing any rate limiter checks during cleanup
			mu.Lock()

			// Delete client from map if not seen within last 3 mins
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Extract IP address from request
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(rw, r, err)
			return
		}

		// Lock mutex preventing concurrent execution
		mu.Lock()

		// Check if IP address already in map, if not add to map w/ new rate limiter
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(2, 4)}
		}

		// Update last seen time for current client
		clients[ip].lastSeen = time.Now()

		// Call Allow() method on rate limiter for current IP address, unlock mutex & send 429 if request not allowed
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(rw, r)
			return
		}

		// Unlock mutex before calling next handler
		mu.Unlock()

		next.ServeHTTP(rw, r)
	})
}
