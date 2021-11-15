package main

import (
	"fmt"
	"net/http"
)

// Generic helper method for logging error messages
func (app *application) logError(r *http.Request, err error) {
	app.logger.Println(err)
}

// Generic helper method for sending JSON-formatted error messages to client
func (app *application) errorResponse(rw http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := envelope{"error": message}

	err := app.writeJSON(rw, status, env, nil)
	if err != nil {
		app.logError(r, err)
		rw.WriteHeader(500)
	}
}

// Method for unexpected problems at runtime
func (app *application) serverErrorResponse(rw http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	message := "The server encountered a problem and could not process your request"
	app.errorResponse(rw, r, http.StatusInternalServerError, message)
}

// 404 Not Found
func (app *application) notFoundResponse(rw http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(rw, r, http.StatusNotFound, message)
}

// 405 Method Not Allowed
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// 400 Bad Request
func (app *application) badRequestResponse(rw http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(rw, r, http.StatusBadRequest, err.Error())
}
