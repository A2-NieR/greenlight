package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/mongo"
)

func (app *application) registerUserHandler(rw http.ResponseWriter, r *http.Request) {
	// Anonymous struct to hold expected data from req body
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Parse req body into anonymous struct
	err := app.readJSON(rw, r, &input)
	if err != nil {
		app.badRequestResponse(rw, r, err)
		return
	}

	// Copy data from req body into new User struct
	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	// Generate and store hashed & plaintext passwords
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}

	v := validator.New()

	// Validate user struct, return error messages to client if checks fail
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(rw, r, v.Errors)
		return
	}

	// Insert user data into DB
	id, err := app.models.User.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateName):
			v.AddError("email", "a user with this name already exists")
			app.failedValidationResponse(rw, r, v.Errors)
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(rw, r, v.Errors)
		default:
			app.serverErrorResponse(rw, r, err)
		}
		return
	}

	// Generate new activation token for user
	token, err := app.models.Token.New(id, time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}

	// Background welcome-email routine
	app.background(func() {
		// Map to hold the data for activation email
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          id,
			"name":            user.Name,
		}
		// Send welcome mail using registered user data
		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	// Write & send 201 JSON response with user data
	err = app.writeJSON(rw, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}
}

func (app *application) activateUserHandler(rw http.ResponseWriter, r *http.Request) {
	// Parse plaintext activation token from req body
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := app.readJSON(rw, r, &input)
	if err != nil {
		app.badRequestResponse(rw, r, err)
		return
	}

	// Validate plaintext token
	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(rw, r, v.Errors)
		return
	}

	// Get user details associated with token
	token, err := app.models.Token.Get(data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(rw, r, v.Errors)
		default:
			app.serverErrorResponse(rw, r, err)
		}
		return
	}

	user, err := app.models.User.GetForToken(token)
	if err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			v.AddError("token", "user for token not found")
			app.failedValidationResponse(rw, r, v.Errors)
		default:
			app.serverErrorResponse(rw, r, err)
		}
		return
	}

	// Update user's activation status
	user.Activated = true

	err = app.models.User.Update(user, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(rw, r)
		default:
			app.serverErrorResponse(rw, r, err)
		}
		return
	}

	// On success delete all activation tokens for user
	err = app.models.Token.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}

	// Send updated user details to client
	err = app.writeJSON(rw, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}
}
