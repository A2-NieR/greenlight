package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/mongo"
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

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			app.notFoundResponse(rw, r)
		default:
			app.serverErrorResponse(rw, r, err)
		}

		return
	}

	err = app.writeJSON(rw, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}
}

func (app *application) updateMovieHandler(rw http.ResponseWriter, r *http.Request) {
	// Extract ID from URL
	id := app.readIDParam(r)

	// Get existing movie record from db
	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			app.notFoundResponse(rw, r)
		default:
			app.serverErrorResponse(rw, r, err)
		}

		return
	}

	// Input struct to hold expected data from client
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	// Read JSON request body into input struct
	err = app.readJSON(rw, r, &input)
	if err != nil {
		app.badRequestResponse(rw, r, err)
		return
	}

	// Copy values from req body to resp. fields of movie record
	movie.Title = input.Title
	movie.Year = input.Year
	movie.Runtime = input.Runtime
	movie.Genres = input.Genres

	// Validate updated movie record
	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(rw, r, v.Errors)
		return
	}

	// Send update request with new record
	err = app.models.Movies.Update(movie, id)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}

	// Write updated data in JSON response
	err = app.writeJSON(rw, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}
}
