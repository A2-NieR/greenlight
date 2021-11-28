package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/mongo"
)

func (app *application) createAuthenticationTokenHandler(rw http.ResponseWriter, r *http.Request) {
	// Parse email & password from req body
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(rw, r, &input)
	if err != nil {
		app.badRequestResponse(rw, r, err)
		return
	}

	// Validate email & password
	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(rw, r, v.Errors)
		return
	}

	// Get user via email
	user, err := app.models.User.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			app.invalidCredentialsResponse(rw, r)
		default:
			app.serverErrorResponse(rw, r, err)
		}
		return
	}

	// Check if password matches
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}

	if !match {
		app.invalidCredentialsResponse(rw, r)
		return
	}

	// Generate new token
	token, err := app.models.Token.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}

	// Encode token to JSON & send with 201
	err = app.writeJSON(rw, http.StatusCreated, envelope{"authentication_token": token}, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}
}
