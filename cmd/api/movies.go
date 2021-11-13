package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
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
		app.logger.Println(err)
		http.Error(rw, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}
