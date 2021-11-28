package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/mongo"
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
		if app.config.limiter.enabled {
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
		}

		next.ServeHTTP(rw, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Add "Vary: Authorization" header to response, indicates to caches the response may vary based on value of Authorization header in request
		rw.Header().Add("Vary", "Authorization")

		// Retrieve value of Authorization header from req
		authorizationHeader := r.Header.Get("Authorization")

		// Add anonymous user to req context if no Authorization header found
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(rw, r)
			return
		}

		// Split header & check format
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(rw, r)
			return
		}

		// Extract auth token from header parts
		token := headerParts[1]

		// Validate token
		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(rw, r)
			return
		}

		// Retrieve user details associated with auth token
		userID, err := app.models.Token.Get(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, mongo.ErrNoDocuments):
				app.invalidAuthenticationTokenResponse(rw, r)
			default:
				app.serverErrorResponse(rw, r, err)
			}
			return
		}

		user, err := app.models.User.GetForToken(userID)
		if err != nil {
			switch {
			case errors.Is(err, mongo.ErrNoDocuments):
				app.invalidAuthenticationTokenResponse(rw, r)
			default:
				app.serverErrorResponse(rw, r, err)
			}
			return
		}

		// Add user info to req context
		r = app.contextSetUser(r, user)

		// Call next handler in chain
		next.ServeHTTP(rw, r)
	})
}

// Check if user is anonymous
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(rw, r)
			return
		}

		next.ServeHTTP(rw, r)
	})
}

// Check if user is authenticated AND activated
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		// Check if user is activated
		if !user.Activated {
			app.inactiveAccountResponse(rw, r)
			return
		}

		next.ServeHTTP(rw, r)
	})

	// Wrap fn with requireAuthenticatedUser() middleware before return
	return app.requireAuthenticatedUser(fn)
}
