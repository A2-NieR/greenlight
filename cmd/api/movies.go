package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// Add createMovieHandler for "POST /v1/movies" endpoint
func (app *application) createMovieHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(rw, "Create a new movie")
}

// Add showMovieHandler for "GET /v1/movies/:id" endpoint + retrieve interpolated "id" parameter from current URL
func (app *application) showMovieHandler(rw http.ResponseWriter, r *http.Request) {
	// Store any interpolated URL params in request context during parsing, retrieve a slice containing params names & values
	params := httprouter.ParamsFromContext(r.Context())

	// Get value of "id" parameter from slice, convert returned string to base 10 int (with bitsize of 64)
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(rw, r)
		return
	}

	fmt.Fprintf(rw, "Show details of movie %d", id)
}
