package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
)

// Add createMovieHandler for "POST /v1/movies" endpoint
func (app *application) createMovieHandler(rw http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	err := app.readJSON(rw, r, &input)
	if err != nil {
		app.badRequestResponse(rw, r, err)
		return
	}

	fmt.Fprintf(rw, "%+v\n", input)
}

// Add showMovieHandler for "GET /v1/movies/:id" endpoint + retrieve interpolated "id" parameter from current URL
func (app *application) showMovieHandler(rw http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(rw, r)
		return
	}

	movie := data.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "Casablanca",
		Runtime:   102,
		Genres:    []string{"drama", "romance", "war"},
		Version:   1,
	}

	err = app.writeJSON(rw, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}
}
