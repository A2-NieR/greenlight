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
		Title   *string       `json:"title"`
		Year    *int32        `json:"year"`
		Runtime *data.Runtime `json:"runtime"`
		Genres  []string      `json:"genres"`
	}

	// Read JSON request body into input struct
	err = app.readJSON(rw, r, &input)
	if err != nil {
		app.badRequestResponse(rw, r, err)
		return
	}

	// Check if values are provided in JSON request body
	if input.Title != nil {
		movie.Title = *input.Title
	}

	if input.Year != nil {
		movie.Year = *input.Year
	}

	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}

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

func (app *application) deleteMovieHandler(rw http.ResponseWriter, r *http.Request) {
	id := app.readIDParam(r)

	err := app.models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			app.notFoundResponse(rw, r)
		default:
			app.serverErrorResponse(rw, r, err)
		}
		return
	}

	err = app.writeJSON(rw, http.StatusOK, envelope{"message": "movie successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}

}

func (app *application) listMoviesHandler(rw http.ResponseWriter, r *http.Request) {
	// Struct to hold expected values from req query string
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}
	// Initialize new validator
	v := validator.New()

	// Get url.Values map containing query string data
	qs := r.URL.Query()

	// Extract title & genre string values
	input.Title = app.readString(qs, "title", "")
	input.Genres = app.readCSV(qs, "genres", []string{})

	// Extract page & pagesize string values as integers
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	// Extract sort query string value
	input.Filters.Sort = app.readString(qs, "sort", "id")

	// Supported sort values
	input.Filters.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	// Check validator instance for any errors
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(rw, r, v.Errors)
		return
	}

	fmt.Fprintf(rw, "%+v\n", input)
}
