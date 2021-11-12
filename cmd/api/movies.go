package main

import (
	"fmt"
	"net/http"
)

// Add createMovieHandler for "POST /v1/movies" endpoint
func (app *application) createMovieHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(rw, "Create a new movie")
}

// Add showMovieHandler for "GET /v1/movies/:id" endpoint + retrieve interpolated "id" parameter from current URL
func (app *application) showMovieHandler(rw http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		http.NotFound(rw, r)
		return
	}

	fmt.Fprintf(rw, "Show details of movie %d", id)
}
