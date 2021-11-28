package main

import (
	"context"
	"net/http"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
)

type contextKey string

// Key for getting & setting user info in req context
const userContextKey = contextKey("user")

// Return a new copy of req with provided User struct added to context
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.Clone(ctx)
}

// Retrieve User struct from req context
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return user
}
