package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	// Convert notFoundResponse() helper to a http.Handler using http.HandlerFunc() adapter & set as custom error handler for 404
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	// Same for 404
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// Register relevant methods, URL patterns & handler functions for endpoints using HandlerFunc() method
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	// Movies CRUD endpoints
	router.HandlerFunc(http.MethodGet, "/v1/movies", app.requirePermission("movies:read", app.listMoviesHandler))
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.requirePermission("movies:write", app.createMovieHandler))
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.requirePermission("movies:read", app.showMovieHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.requirePermission("movies:write", app.updateMovieHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.requirePermission("movies:write", app.deleteMovieHandler))

	// User endpoints
	router.HandlerFunc(http.MethodPost, "/v1/user", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/user/activate", app.activateUserHandler)

	// Token endpoints
	router.HandlerFunc(http.MethodPost, "/v1/token/authentication", app.createAuthenticationTokenHandler)

	// Debug/Metric endpoints
	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
