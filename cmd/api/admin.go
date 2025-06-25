package main

import (
	"fmt"
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
	worker := getUserFromContext(r)
	if worker == nil || worker.Role != "worker" {
		app.unauthorizedErrorResponse(w, r, fmt.Errorf("you dont have permissions to access this"))
		return
	}
	workerID := worker.ID

	_, month, _ := selectedDay.Date()
	data, err := app.store.TimeSlots.GetBookedNumberForAMonth(r.Context(), int(month), workerID)

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

// ako se u body json ne stavi nista, onda ce automatski uzeti danasnji dan
func (app *application) getBookedDates(w http.ResponseWriter, r *http.Request) {
	var selectedDayPayload selectedDayPayload
	if err := readJSON(w, r, &selectedDayPayload); err != nil {
		selectedDayPayload.Day = time.Now().Format(time.DateOnly)
	}

	selectedDay, err := time.Parse(time.DateOnly, selectedDayPayload.Day)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	worker := getUserFromContext(r)
	if worker == nil || worker.Role != "worker" {
		app.unauthorizedErrorResponse(w, r, fmt.Errorf("you dont have permissions to access this"))
		return
	}
	workerID := worker.ID

	slots, err := app.store.TimeSlots.GetSlots(r.Context(), selectedDay, workerID, true)
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

	if err := app.jsonResponse(w, http.StatusOK, slots); err != nil {
		app.internalServerError(w, r, err)
	}
}
