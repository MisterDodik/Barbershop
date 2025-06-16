package main

import "net/http"

func (app *application) getHealthHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":  "ok",
		"env":     "to-do",
		"version": version,
	}
	if err := app.jsonResponse(w, http.StatusOK, data); err != nil {
		writeJSONError(w, http.StatusBadRequest, "something went wrong")
	}
}
