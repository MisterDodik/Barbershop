package main

import (
	"net/http"
	"time"

	"github.com/MisterDodik/Barbershop/internal/store"
)

func (app *application) getCalendarValues(w http.ResponseWriter, r *http.Request) {

	var selectedDayPayload selectedDayPayload
	if err := readJSON(w, r, &selectedDayPayload); err != nil {
		selectedDayPayload.Day = time.Now().Format(time.DateOnly)
	}

	selectedDay, err := time.Parse(time.DateOnly, selectedDayPayload.Day)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	_, month, _ := selectedDay.Date()
	data, err := app.store.TimeSlots.GetBookedNumberForAMonth(r.Context(), int(month))

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
