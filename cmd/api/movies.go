package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(rw, r, v.Errors)
		return
	}

	id, err := app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}
	movie.ID = id

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%s", id))

	err = app.writeJSON(rw, http.StatusCreated, envelope{"movie": movie}, headers)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}
}

// Add showMovieHandler for "GET /v1/movies/:id" endpoint + retrieve interpolated "id" parameter from current URL
func (app *application) showMovieHandler(rw http.ResponseWriter, r *http.Request) {
	id := app.readIDParam(r)
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		app.notFoundResponse(rw, r)
		return
	}

	movie := data.Movie{
		OID:       oid,
		ID:        oid.Hex(),
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
