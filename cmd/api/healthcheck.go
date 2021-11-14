package main

import (
	"net/http"
)

func (app *application) healthcheckHandler(rw http.ResponseWriter, r *http.Request) {
	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	err := app.writeJSON(rw, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}
}
