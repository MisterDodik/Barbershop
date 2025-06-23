package main

import (
	"net/http"

	"github.com/MisterDodik/Barbershop/internal/store"
)

func (app *application) getCalendarValues(w http.ResponseWriter, r *http.Request) {

	month := 1

	data, err := app.store.TimeSlots.GetBookedNumberForAMonth(r.Context(), month)

	if err != nil {
		switch err {
		case store.Error_NotFound:
			app.notFoundResponse(w, r, err)
			return
		default:
			app.internalServerError(w, r, err)
			return
		}
	}

	if err := app.jsonResponse(w, http.StatusOK, data); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
