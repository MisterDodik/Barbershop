package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MisterDodik/Barbershop/internal/store"
	"github.com/go-chi/chi/v5"
)

type selectedDayPayload struct {
	Day string `json:"day"`
}

// ako se u body json ne stavi nista, onda ce automatski uzeti danasnji dan
func (app *application) getAvailableDates(w http.ResponseWriter, r *http.Request) {
	var selectedDayPayload selectedDayPayload
	if err := readJSON(w, r, &selectedDayPayload); err != nil {
		selectedDayPayload.Day = time.Now().Format(time.DateOnly)
	}

	selectedDay, err := time.Parse(time.DateOnly, selectedDayPayload.Day)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	slots, err := app.store.TimeSlots.GetFreeSlots(r.Context(), selectedDay)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, slots); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) bookAppointment(w http.ResponseWriter, r *http.Request) {
	slotIDstr := chi.URLParam(r, "slotID")
	slotID, err := strconv.ParseInt(slotIDstr, 10, 64)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.store.TimeSlots.Book(r.Context(), slotID); err != nil {
		switch err {
		case store.Error_NotFound:
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, nil); err != nil {
		app.internalServerError(w, r, err)
	}
}
